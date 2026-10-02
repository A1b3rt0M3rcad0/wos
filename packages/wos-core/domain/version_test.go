package domain_test

import (
	"math"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestVersionLifecycle(t *testing.T) {
	if err := domain.InitialVersion.Validate(); err != nil {
		t.Fatalf("InitialVersion.Validate() error = %v", err)
	}

	next, err := domain.InitialVersion.Next()
	if err != nil {
		t.Fatalf("InitialVersion.Next() error = %v", err)
	}
	if next != 2 {
		t.Fatalf("InitialVersion.Next() = %d, want 2", next)
	}

	if _, err := domain.Version(0).Next(); err == nil {
		t.Fatal("Version(0).Next() unexpectedly succeeded")
	}
	if _, err := domain.Version(math.MaxUint64).Next(); err == nil {
		t.Fatal("max Version.Next() unexpectedly succeeded")
	}
}

func TestOutcomeRevisionStartsAtZero(t *testing.T) {
	next, err := domain.InitialOutcomeRevision.Next()
	if err != nil {
		t.Fatalf("InitialOutcomeRevision.Next() error = %v", err)
	}
	if next != 1 {
		t.Fatalf("InitialOutcomeRevision.Next() = %d, want 1", next)
	}
}
