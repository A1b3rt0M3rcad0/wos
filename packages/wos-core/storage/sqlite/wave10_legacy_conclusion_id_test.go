package sqlite

import (
	"testing"
	"time"
)

func TestLegacyConclusionPublicIDIsStableUUIDv7(t *testing.T) {
	recordedAt := time.Date(2026, 10, 2, 21, 30, 0, 0, time.UTC)
	first, err := legacyConclusionPublicID("legacy-storage-hash", recordedAt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := legacyConclusionPublicID("legacy-storage-hash", recordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("legacy public id is not stable: %s != %s", first, second)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("legacy public id is not valid UUIDv7: %v", err)
	}
	other, err := legacyConclusionPublicID("different-storage-hash", recordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if other == first {
		t.Fatal("different legacy storage identities produced the same public id")
	}
}
