package woscli

import (
	"bytes"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestrictedYAMLRejectsDangerousOrAmbiguousForms(t *testing.T) {
	tests := []string{"a: 1\na: 2\n", "a: &x [1]\nb: *x\n", "a: !custom value\n", "a: !!str value\n", "a: {<<: {x: 1}}\n", "[a,b]\n", "a: 1\n---\nb: 2\n", "1: value\n", "a: .nan\n", "a: 0x10\n", "a: 01\n", "a: True\n", string([]byte{'a', ':', ' ', 0xff}), "a: " + strings.Repeat("x", 65537)}
	nested := "0"
	for i := 0; i < 18; i++ {
		nested = "[" + nested + "]"
	}
	tests = append(tests, "a: "+nested, "a: "+strings.Repeat("x", MaxDocumentBytes))
	for _, raw := range tests {
		if _, err := YAMLJSON([]byte(raw)); err == nil {
			t.Errorf("accepted %q", raw[:min(80, len(raw))])
		}
	}
}
func TestRestrictedYAMLExactTokensAndSemanticFormatting(t *testing.T) {
	type doc struct {
		Token domain.FencingToken `json:"fencing_token"`
		Label string              `json:"label"`
	}
	for _, raw := range []string{"fencing_token: '9007199254740993'\nlabel: on\n", "# comment\nlabel: on\nfencing_token: \"9007199254740993\"\n"} {
		var v doc
		if err := DecodeDocument([]byte(raw), &v); err != nil {
			t.Fatal(err)
		}
		if uint64(v.Token) != 9007199254740993 || v.Label != "on" {
			t.Fatal("YAML coercion")
		}
		encoded, err := EncodeDocument(v)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip doc
		if err = DecodeDocument(encoded, &roundtrip); err != nil || roundtrip != v {
			t.Fatalf("roundtrip %s %v", encoded, err)
		}
	}
	for _, raw := range []string{"fencing_token: 9007199254740993\nlabel: x\n", "fencing_token: '18446744073709551616'\nlabel: x\n", "fencing_token: '1'\nunknown: x\n"} {
		var v doc
		if err := DecodeDocument([]byte(raw), &v); err == nil {
			t.Fatal("invalid typed token/field accepted")
		}
	}
	first, _ := YAMLJSON([]byte("label: first\nitems: [one, two]\n"))
	second, _ := YAMLJSON([]byte("# c\nitems:\n  - one\n  - two\nlabel: first\n"))
	a, _ := domain.SemanticDigest(json.RawMessage(first))
	b, _ := domain.SemanticDigest(json.RawMessage(second))
	if a != b {
		t.Fatal("formatting affected digest")
	}
}
func TestWorkspaceRefusesEscapesAndPreservesAtomicFiles(t *testing.T) {
	root := t.TempDir()
	w, err := OpenWorkspace(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	for _, path := range []string{"../outside", "/tmp/outside", "..\\outside", "C:\\outside"} {
		if err = w.AtomicWrite(path, []byte("bad")); err == nil {
			t.Fatal("path escaped")
		}
	}
	if err = w.AtomicWrite(".wos/value.json", []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err = w.AtomicWrite(".wos/value.json", []byte("second")); err != nil {
		t.Fatal(err)
	}
	raw, err := w.Read(".wos/value.json")
	if err != nil || !bytes.Equal(raw, []byte("second")) {
		t.Fatal("atomic replacement failed")
	}
	if err = os.Symlink(t.TempDir(), filepath.Join(root, "link")); err == nil {
		if err = w.AtomicWrite("link/value", []byte("bad")); err == nil {
			t.Fatal("symlink write accepted")
		}
	}
	unlock, err := w.Lock(".wos/locked")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Lock(".wos/locked"); err == nil {
		t.Fatal("concurrent lock acquired")
	}
	unlock()
	release, err := w.Lock(".wos/locked")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err = w.AtomicWrite(".wos/config.yaml", []byte("a: 1")); err != nil {
		t.Fatal(err)
	}
	if err = w.AtomicWrite(".wos/config.yml", []byte("a: 1")); err != nil {
		t.Fatal(err)
	}
	if _, err = w.DocumentPath(".wos/config"); err == nil {
		t.Fatal("ambiguous YAML extensions accepted")
	}
}
func FuzzYAMLJSON(f *testing.F) {
	for _, seed := range []string{"a: 1\n", "a: [true, false]\n", "a: &x foo\n", "a: '9007199254740993'\n"} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := YAMLJSON(raw)
		if err == nil && !json.Valid(out) {
			t.Fatal("invalid JSON produced")
		}
	})
}
