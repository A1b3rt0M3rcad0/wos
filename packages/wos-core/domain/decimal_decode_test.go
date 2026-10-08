package domain

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCounterDecodePreservesV1EncodingAndExactV2Values(t *testing.T) {
	for _, raw := range []string{`9007199254740993`, `"9007199254740993"`, `18446744073709551615`, `"18446744073709551615"`} {
		var counter Version
		if err := json.Unmarshal([]byte(raw), &counter); err != nil {
			t.Fatal(raw, err)
		}
		if counter != Version(9007199254740993) && counter != Version(math.MaxUint64) {
			t.Fatal("counter rounded")
		}
		encoded, err := json.Marshal(counter)
		if err != nil || encoded[0] == '"' {
			t.Fatal("historical numeric encoding changed", err)
		}
	}
	for _, raw := range []string{`"01"`, `"+1"`, `"1e3"`, `"18446744073709551616"`, `-1`, `1.5`, `null`} {
		var counter Version
		if json.Unmarshal([]byte(raw), &counter) == nil {
			t.Fatal("accepted noncanonical counter", raw)
		}
	}
	var tagged struct {
		Version Version `json:"version,string"`
	}
	if err := json.Unmarshal([]byte(`{"version":"18446744073709551615"}`), &tagged); err != nil || tagged.Version != Version(math.MaxUint64) {
		t.Fatal("quoted signed binding cannot decode exactly", err)
	}
}

func TestExactFenceDecodingKeepsHistoricalPartialDecodeSemantics(t *testing.T) {
	work := WorkItem{Title: "preserved", LastFencingToken: 1}
	if err := json.Unmarshal([]byte(`{"last_fencing_token":"18446744073709551615"}`), &work); err != nil || work.Title != "preserved" || work.LastFencingToken != math.MaxUint64 {
		t.Fatal("exact Task fence decode changed unrelated fields", err)
	}
	lease := WorkLease{ClaimID: MustParseID("0199a555-0000-7000-8000-000000000001"), PrincipalID: "preserved", Actor: ActorRef{Kind: ActorKindAgent, Provider: "test", ID: "worker"}, FencingToken: 1}
	if err := json.Unmarshal([]byte(`{"fencing_token":"9007199254740993"}`), &lease); err != nil || lease.PrincipalID != "preserved" || lease.FencingToken != 9007199254740993 {
		t.Fatal("exact lease fence decode changed unrelated fields", err)
	}
	if err := json.Unmarshal([]byte(`{"fencing_token":7}`), &lease); err != nil || lease.FencingToken != 7 {
		t.Fatal("historical lease fence cannot decode", err)
	}
	raw, err := json.Marshal(lease)
	if err != nil {
		t.Fatal(err)
	}
	var historical struct {
		Fence uint64 `json:"fencing_token"`
	}
	if err = json.Unmarshal(raw, &historical); err != nil || historical.Fence != 7 {
		t.Fatal("historical lease encoding changed", err)
	}
}
