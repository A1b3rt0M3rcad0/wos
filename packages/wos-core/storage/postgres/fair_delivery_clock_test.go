package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestDeliveryFleetAuthorityUsesDatabaseTimeDespiteHostSkew(t *testing.T) {
	requirePostgres(t)
	store, _, _ := fairDeliveryFixture(t)
	ctx := context.Background()
	policy := ports.DeliveryClaimPolicy{DatabaseTime: true, MaxInflightPerNamespace: 1, SecretRefs: []string{"tenant-1"}}
	before := time.Now()
	v, err := store.ClaimDeliveryWithPolicy(ctx, before.Add(24*time.Hour), testID("0199df10-0000-7000-8000-000000000001"), time.Minute, policy)
	if err != nil || v == nil {
		t.Fatalf("claim: %+v %v", v, err)
	}
	if v.LeaseUntil.Before(before.Add(50*time.Second)) || v.LeaseUntil.After(time.Now().Add(70*time.Second)) {
		t.Fatal("host clock changed fleet lease expiry", v.LeaseUntil)
	}
	other, err := store.ClaimDeliveryWithPolicy(ctx, before.Add(48*time.Hour), testID("0199df10-0000-7000-8000-000000000002"), time.Minute, policy)
	if err != nil || other != nil {
		t.Fatalf("skewed host stole live capacity: %+v %v", other, err)
	}
	if err = store.FinishDeliveryUsingDatabaseTime(ctx, *v, "transport_error", false, time.Minute); err != nil {
		t.Fatal(err)
	}
	var next int64
	if err = store.db.QueryRowContext(ctx, `SELECT next_attempt FROM integration_deliveries WHERE id=$1`, v.ID).Scan(&next); err != nil {
		t.Fatal(err)
	}
	if at := decodeTime(next); at.Before(before.Add(50*time.Second)) || at.After(time.Now().Add(70*time.Second)) {
		t.Fatal("retry schedule used worker host time", at)
	}
	// A persisted expired lease cannot be acknowledged by the authoritative ACK.
	other, err = store.ClaimDeliveryWithPolicy(ctx, before, testID("0199df10-0000-7000-8000-000000000003"), time.Minute, policy)
	if err != nil || other == nil {
		t.Fatalf("claim after releasing capacity: %+v %v", other, err)
	}
	if _, err = store.db.ExecContext(ctx, `UPDATE integration_deliveries SET lease_until=1 WHERE id=$1`, other.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.FinishDeliveryUsingDatabaseTime(ctx, *other, "http_200", true, 0); err == nil {
		t.Fatal("expired authoritative lease accepted ACK")
	}
}
