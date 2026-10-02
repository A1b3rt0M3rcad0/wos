package ports_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDGenerator struct{ id domain.ID }

func (g fixedIDGenerator) NewID() (domain.ID, error) { return g.id, nil }

func TestPublicRuntimePortsAreImplementable(t *testing.T) {
	var _ ports.Clock = fixedClock{}
	var _ ports.IDGenerator = fixedIDGenerator{}

	now := time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)
	clock := fixedClock{now: now}
	if got := clock.Now(); !got.Equal(now) {
		t.Fatalf("Clock.Now() = %s, want %s", got, now)
	}

	id := domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	generator := fixedIDGenerator{id: id}
	got, err := generator.NewID()
	if err != nil {
		t.Fatalf("IDGenerator.NewID() error = %v", err)
	}
	if got != id {
		t.Fatalf("IDGenerator.NewID() = %q, want %q", got, id)
	}
}
