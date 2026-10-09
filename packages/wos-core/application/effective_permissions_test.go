package application_test

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"testing"
	"time"
)

type presentationAuthorizer struct {
	denied bool
	fail   bool
	calls  []ports.AuthorizationRequest
}

func (a *presentationAuthorizer) Authorize(_ context.Context, r ports.AuthorizationRequest) error {
	a.calls = append(a.calls, r)
	if a.fail {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "store unavailable")
	}
	if a.denied || r.Permission != ports.PermissionStateRead {
		return domain.NewError(domain.ErrorCodeForbidden, "not granted")
	}
	return nil
}
func TestEffectivePermissionsAreScopedHintsNotAuthority(t *testing.T) {
	a := &presentationAuthorizer{}
	s, err := application.NewAuthorizedService(memory.New(), fixedClock{time.Now()}, &sequenceIDs{}, a)
	if err != nil {
		t.Fatal(err)
	}
	scope := domain.Scope{NamespaceID: id("0199ec00-0000-7000-8000-000000000001"), OutcomeID: id("0199ec00-0000-7000-8000-000000000002")}
	ctx := application.WithIdentity(context.Background(), application.Identity{NamespaceID: scope.NamespaceID, PrincipalID: "viewer"})
	got, err := s.EffectivePermissions(ctx, scope)
	if err != nil || len(got) != 1 || got[0] != ports.PermissionStateRead {
		t.Fatalf("viewer: %v %v", got, err)
	}
	for _, r := range a.calls {
		if r.NamespaceID != scope.NamespaceID || r.OutcomeID != scope.OutcomeID || r.PrincipalID != "viewer" {
			t.Fatal("unscoped permission query")
		}
	}
	a.denied = true
	if _, err = s.EffectivePermissions(ctx, scope); err == nil {
		t.Fatal("revoked read succeeded")
	}
	a.denied = false
	a.fail = true
	if _, err = s.EffectivePermissions(ctx, scope); err == nil {
		t.Fatal("storage error was hidden")
	}
	if _, err = s.EffectivePermissions(context.Background(), scope); err == nil {
		t.Fatal("unauthenticated hints returned")
	}
}
