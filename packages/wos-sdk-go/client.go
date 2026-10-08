// Package wossdk is an HTTP client for WOS. Mutations never retry automatically.
package wossdk

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Prefix  string
	Token   string
	HTTP    *http.Client
}

func New(base, token string, client *http.Client) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid WOS URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	safeClient := *client
	safeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{BaseURL: strings.TrimRight(base, "/"), Prefix: "/api/v1", Token: token, HTTP: &safeClient}, nil
}

type Error struct {
	Status        int
	Code          string
	Message       string
	Retryable     bool
	CorrelationID string
}

func (e *Error) Error() string { return fmt.Sprintf("WOS %s (%d): %s", e.Code, e.Status, e.Message) }

type CommandResult[T any] struct {
	Value            T                      `json:"value"`
	OutcomeRevision  domain.OutcomeRevision `json:"outcome_revision"`
	CommandID        domain.ID              `json:"command_id"`
	IdempotentReplay bool                   `json:"idempotent_replay"`
	ResultOmitted    bool                   `json:"result_omitted"`
}

func NewIdempotencyKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (c *Client) do(ctx context.Context, method, path, key string, input, out any) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+c.Prefix+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (256<<10)+1))
	if err != nil {
		return err
	}
	if len(raw) > 256<<10 {
		return fmt.Errorf("WOS response exceeds 256 KiB")
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return &Error{Status: resp.StatusCode, Code: "transport_redirect", Message: "redirect refused; restore the exact trusted destination"}
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error struct {
				Code      string `json:"code"`
				Message   string `json:"message"`
				Retryable bool   `json:"retryable"`
			}
		}
		json.Unmarshal(raw, &e)
		return &Error{Status: resp.StatusCode, Code: e.Error.Code, Message: e.Error.Message, Retryable: e.Error.Retryable, CorrelationID: resp.Header.Get("X-Correlation-ID")}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}
func command[T any](ctx context.Context, c *Client, name, key string, cmd any) (CommandResult[T], error) {
	var result CommandResult[T]
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return result, err
	}
	raw, err := commands.Encode(cmd)
	if err != nil {
		return result, err
	}
	err = c.do(ctx, "POST", "/commands/"+name, key, map[string]any{"command": json.RawMessage(raw)}, &result)
	return result, err
}
func scopePath(scope domain.Scope) string {
	return "/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String()
}
func (c *Client) GetContinuity(ctx context.Context, scope domain.Scope, limit int) (application.ContinuitySnapshot, error) {
	var v application.ContinuitySnapshot
	err := c.do(ctx, "GET", scopePath(scope)+fmt.Sprintf("/continuity?limit=%d", limit), "", nil, &v)
	return v, err
}
func (c *Client) GetContinuitySection(ctx context.Context, scope domain.Scope, section string, limit int, cursor string) (application.SectionPage, error) {
	var v application.SectionPage
	q := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", scopePath(scope)+"/continuity/"+url.PathEscape(section)+"?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) SearchOutcomes(ctx context.Context, namespace domain.ID, text string, limit int, cursor string) (application.OutcomePage, error) {
	var v application.OutcomePage
	q := url.Values{"text": {text}, "limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", "/namespaces/"+namespace.String()+"/outcomes?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) GetWorkContext(ctx context.Context, scope domain.Scope, id domain.ID) (application.WorkContext, error) {
	var v application.WorkContext
	err := c.do(ctx, "GET", scopePath(scope)+"/work-items/"+id.String()+"/context", "", nil, &v)
	return v, err
}
