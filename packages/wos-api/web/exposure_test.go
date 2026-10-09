package web

import (
	"encoding/json"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
)

// A new public operation requires an explicit product decision. No name-prefix
// heuristic can silently expose it in a human or signed execution surface.
func TestEveryCommandHasExplicitExposure(t *testing.T) {
	raw, err := assets.ReadFile("assets/command-exposure.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory map[string]struct {
		Exposure string `json:"exposure"`
		Journey  string `json:"journey"`
	}
	if err := json.Unmarshal(raw, &inventory); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"H": true, "C": true, "A": true, "L": true, "S": true, "I": true}
	for _, d := range commands.NewCatalog(nil, nil).Descriptors() {
		v, ok := inventory[d.Name]
		if !ok || !allowed[v.Exposure] || v.Journey == "" {
			t.Errorf("missing explicit exposure for %s", d.Name)
		}
		delete(inventory, d.Name)
	}
	for name := range inventory {
		t.Errorf("obsolete command exposure: %s", name)
	}
}
