package integration

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestWebhookSignatureAndDestinationPolicy(t *testing.T) {
	secret := strings.Repeat("s", 40)
	now := time.Unix(1791220000, 0)
	timestamp := "1791220000"
	body := []byte(`{"schema_version":1,"signal_type":"product.ready"}`)
	signature := Signature(secret, timestamp, "event-1", body)
	if err := VerifySignature(secret, timestamp, "event-1", signature, body, now); err != nil {
		t.Fatal(err)
	}
	for _, v := range []struct {
		secret string
		body   []byte
		now    time.Time
	}{{"wrong", body, now}, {secret, []byte("altered"), now}, {secret, body, now.Add(6 * time.Minute)}} {
		if VerifySignature(v.secret, timestamp, "event-1", signature, v.body, v.now) == nil {
			t.Fatal("accepted invalid signature or timestamp")
		}
	}
	for _, raw := range []string{"127.0.0.1", "::1", "10.0.0.1", "192.168.0.1", "169.254.169.254", "0.0.0.0"} {
		if permittedIP(net.ParseIP(raw), false) {
			t.Fatal("accepted private destination", raw)
		}
	}
	if !permittedIP(net.ParseIP("127.0.0.1"), true) {
		t.Fatal("explicit local policy rejected loopback")
	}
}
