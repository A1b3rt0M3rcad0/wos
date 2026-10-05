package application

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
	"unicode/utf8"
)

func FuzzCursorRejectsForeignScope(f *testing.F) {
	f.Add("", "key")
	f.Add("garbage", "chave á")
	ns := domain.MustParseID("0199d095-0000-7000-8000-000000000001")
	other := domain.MustParseID("0199d095-0000-7000-8000-000000000002")
	f.Fuzz(func(t *testing.T, raw, key string) {
		if len(key) > 256 || !utf8.ValidString(key) {
			return
		}
		encoded := encodeCursor(queryCursor{Namespace: ns, Section: "work_items", Key: key, Revision: 1})
		decoded, err := decodeCursor(encoded, ns, "", "work_items", "")
		if err != nil || decoded.Key != key {
			t.Fatalf("round trip: %v", err)
		}
		if _, err = decodeCursor(encoded, other, "", "work_items", ""); err == nil {
			t.Fatal("foreign scope accepted")
		}
		_, _ = decodeCursor(raw, ns, "", "work_items", "")
	})
}
