// Package integration delivers durable signals; it does not execute domain work.
package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Worker struct {
	Store         ports.DeliveryStore
	Clock         ports.Clock
	IDs           ports.IDGenerator
	Secrets       map[string]string
	Client        *http.Client
	allowLoopback bool
}
type Policy struct {
	AllowedHosts  []string
	AllowLoopback bool
}

func New(store ports.DeliveryStore, clock ports.Clock, ids ports.IDGenerator, secrets map[string]string, policy Policy) (*Worker, error) {
	if store == nil || clock == nil || ids == nil {
		return nil, fmt.Errorf("delivery worker dependencies required")
	}
	allowed := map[string]bool{}
	for _, host := range policy.AllowedHosts {
		allowed[strings.ToLower(host)] = true
	}
	dialer := net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || !allowed[strings.ToLower(host)] {
			return nil, fmt.Errorf("destination rejected")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("destination has no addresses")
		}
		for _, entry := range ips {
			if !permittedIP(entry.IP, policy.AllowLoopback) {
				return nil, fmt.Errorf("destination address rejected")
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	return &Worker{Store: store, Clock: clock, IDs: ids, Secrets: secrets, Client: client, allowLoopback: policy.AllowLoopback}, nil
}
func permittedIP(ip net.IP, loopback bool) bool {
	if ip.IsLoopback() {
		return loopback
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsUnspecified() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}
func Signature(secret, timestamp, id string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + id + "."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
func VerifySignature(secret, timestamp, id, signature string, body []byte, now time.Time) error {
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || id == "" || len(id) > 128 || len(body) > 256<<10 {
		return fmt.Errorf("invalid webhook metadata")
	}
	delta := now.Sub(time.Unix(seconds, 0))
	if delta < -5*time.Minute || delta > 5*time.Minute {
		return fmt.Errorf("webhook timestamp outside tolerance")
	}
	expected := Signature(secret, timestamp, id, body)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid webhook signature")
	}
	return nil
}
func (w *Worker) Tick(ctx context.Context) (bool, error) {
	id, err := w.IDs.NewID()
	if err != nil {
		return false, err
	}
	delivery, err := w.Store.ClaimDelivery(ctx, w.Clock.Now().UTC(), id, 30*time.Second)
	if err != nil || delivery == nil {
		return false, err
	}
	result := "destination_configuration_error"
	success := false
	secret, configured := w.Secrets[delivery.SecretRef]
	target, parseErr := url.Parse(delivery.URL)
	localHTTP := false
	if parseErr == nil {
		ip := net.ParseIP(target.Hostname())
		localHTTP = w.allowLoopback && target.Scheme == "http" && (target.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()))
	}
	if configured && len(secret) >= 32 && parseErr == nil && (target.Scheme == "https" || localHTTP) {
		request, err := http.NewRequestWithContext(ctx, "POST", delivery.URL, strings.NewReader(string(delivery.Body)))
		if err == nil {
			timestamp := strconv.FormatInt(w.Clock.Now().Unix(), 10)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-WOS-Event-ID", delivery.IntegrationEventID.String())
			request.Header.Set("X-WOS-Timestamp", timestamp)
			request.Header.Set("X-WOS-Key-ID", delivery.KeyID)
			request.Header.Set("X-WOS-Signature", Signature(secret, timestamp, delivery.IntegrationEventID.String(), delivery.Body))
			response, err := w.Client.Do(request)
			if err == nil {
				_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
				response.Body.Close()
				result = fmt.Sprintf("http_%d", response.StatusCode)
				success = response.StatusCode >= 200 && response.StatusCode < 300 && readErr == nil
			} else {
				result = "transport_error"
				if ctx.Err() != nil {
					result = "interrupted"
				}
			}
		}
	}
	backoff := 2 * time.Second * time.Duration(1<<min(delivery.Attempts-1, 8))
	next := w.Clock.Now().Add(backoff)
	// Recording a cancelled request remains bounded and independent of cancellation;
	// delivery may have reached the receiver, so retry uses the same event identity.
	ackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = w.Store.FinishDelivery(ackCtx, *delivery, w.Clock.Now().UTC(), result, success, next)
	return true, err
}
func (w *Worker) Run(ctx context.Context) error {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			for i := 0; i < 25; i++ {
				worked, err := w.Tick(ctx)
				if err != nil && ctx.Err() == nil {
					code, _ := domain.ErrorCodeOf(err)
					slog.Warn("delivery_worker", "error_code", code)
				}
				if err != nil || !worked {
					break
				}
				if ctx.Err() != nil {
					return nil
				}
			}
		}
	}
}
