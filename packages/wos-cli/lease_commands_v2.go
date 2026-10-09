package woscli

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"strconv"
	"time"
)

func leaseCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: o.args[0] + " " + o.args[1]}
	kind := "execution"
	if o.args[0] == "review" {
		kind = "review"
	}
	ttl := profile.Lease.RequestedTTLSeconds
	if value := o.values["ttl"]; value != "" {
		var e error
		ttl, e = strconv.Atoi(value)
		if e != nil || ttl < 30 || ttl > 3600 {
			return result, usage("TTL must be 30..3600 seconds")
		}
	}
	if o.args[1] == "refresh" {
		return refreshContractV2(ctx, w, profile, client, token, o, kind)
	}
	if o.args[1] == "renew" || o.args[1] == "resume" {
		if len(o.args) != 3 {
			return result, usage("renew/resume requires one contract ID")
		}
		if o.args[1] == "resume" && o.values["ttl"] != "" {
			return result, usage("takeover does not extend expiry; omit --ttl")
		}
		id, e := d.ParseID(o.args[2])
		if e != nil {
			return result, e
		}
		intent, e := prepareLeaseV2(ctx, w, profile, token, identity.Actor, id, kind, o.args[1] == "resume", ttl)
		if e != nil {
			return result, e
		}
		accepted, e := recoverLeaseV2(ctx, w, profile, client, token, intent)
		result.ContractID = id
		result.Committed = !accepted.CommandID.IsZero()
		if e != nil {
			result.RequiresAction = o.args[0] + " recover replays this original lease intention"
			return result, e
		}
		result.Data = map[string]any{"contract_id": id, "command_id": accepted.CommandID, "authority_digest": signing.Digest(accepted.Authority.Payload)}
		return result, nil
	}
	if o.args[1] != "keepalive" || len(o.args) != 2 || o.values["all-active"] != "true" {
		return result, usage("keepalive --all-active --foreground [--once]")
	}
	if o.values["foreground"] != "true" && o.values["once"] != "true" {
		return result, usage("keepalive requires explicit --foreground or --once")
	}
	interval := min(ttl/3, 300)
	if value := o.values["interval"]; value != "" {
		var e error
		interval, e = strconv.Atoi(value)
		if e != nil || interval < 1 || interval > 300 || interval >= ttl {
			return result, usage("keepalive interval must be 1..300 seconds and shorter than TTL")
		}
	}
	for {
		ids, e := contractIDsV2(w, profile.Name)
		if e != nil {
			return result, e
		}
		items := []any{}
		failed := false
		active := 0
		for _, id := range ids {
			file, view, _, _, e := loadContractV2(w, profile, id)
			if e != nil {
				failed = true
				items = append(items, map[string]any{"contract_id": id, "error": "local contract could not be verified"})
				continue
			}
			if view.Authority.ContractKind != kind {
				continue
			}
			if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
				items = append(items, map[string]any{"contract_id": id, "skipped": "final_return_pending_or_accepted"})
				continue
			}
			state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}, Resource: kind, ID: id})
			if e != nil {
				failed = true
				items = append(items, map[string]any{"contract_id": id, "error": "live state unavailable"})
				continue
			}
			if state.Contract == nil || !state.Contract.Current || !state.Contract.LeaseValid {
				items = append(items, map[string]any{"contract_id": id, "skipped": "lease_not_live"})
				continue
			}
			active++
			intent, e := prepareLeaseV2(ctx, w, profile, token, identity.Actor, id, kind, false, ttl)
			if e != nil {
				failed = true
				items = append(items, map[string]any{"contract_id": id, "error": "unresolved intention or local conflict; recover explicitly"})
				continue
			}
			accepted, e := recoverLeaseV2(ctx, w, profile, client, token, intent)
			if !accepted.CommandID.IsZero() {
				result.Committed = true
			}
			item := map[string]any{"contract_id": id, "renewed": e == nil}
			if e != nil {
				failed = true
				item["error"] = "original lease intention retained; recover explicitly"
			}
			items = append(items, item)
		}
		result.Data = map[string]any{"items": items, "partial": failed, "active": active}
		if o.values["once"] == "true" || active == 0 {
			if failed {
				return result, fmt.Errorf("keepalive completed other items with recoverable failures")
			}
			return result, nil
		}
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		case <-timer.C:
		}
	}
}

// Refresh is explicit and read-only remotely. It never takes over/renews a
// contract. A changed Task/case CAS is exposed rather than silently rebased.
func refreshContractV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, o options, kind string) (Output, error) {
	result := Output{Operation: o.args[0] + " refresh"}
	if len(o.args) != 3 {
		return result, usage("refresh requires one contract ID")
	}
	id, e := d.ParseID(o.args[2])
	if e != nil {
		return result, e
	}
	file, view, _, _, e := loadContractV2(w, profile, id)
	if e != nil {
		return result, e
	}
	if view.Authority.ContractKind != kind {
		return result, usage("selected contract kind differs")
	}
	if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
		return result, fmt.Errorf("pending final return pauses refresh")
	}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: kind, ID: id})
	if e != nil {
		return result, e
	}
	if state.Contract == nil || !state.Contract.Current || !state.Contract.LeaseValid || state.Contract.ExecutionID.String() != view.Authority.ExecutionID || state.Contract.FencingToken != view.Authority.FencingToken || state.Contract.WorkItemVersion != file.Local.WorkItemVersion {
		return result, fmt.Errorf("live authority or material CAS changed; explicit reconciliation required")
	}
	authority, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "authority", ContractKind: kind, ID: id})
	if e != nil {
		return result, e
	}
	if authority.Envelope == nil {
		return result, fmt.Errorf("fresh authority proof absent")
	}
	document, e := signing.ToDocument(*authority.Envelope)
	if e != nil {
		return result, e
	}
	var observedCaseVersion signing.Decimal
	if kind == "review" {
		caseState, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "case", ID: d.ID(view.Authority.ReviewCaseID)})
		if e != nil {
			return result, e
		}
		if caseState.ReviewCase == nil {
			return result, fmt.Errorf("review case absent")
		}
		observedCaseVersion = signing.Decimal(caseState.ReviewCase.Version)
		if observedCaseVersion != file.Local.ReviewCaseVersion {
			return result, fmt.Errorf("review material CAS changed; do not rebase")
		}
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		for _, pending := range current.Local.PendingOperations {
			if pending.ContractID != nil && *pending.ContractID == id {
				return fmt.Errorf("unresolved contract intention pauses refresh")
			}
		}
		release, e := w.LockV2(ctx, current.Name, id.String())
		if e != nil {
			return e
		}
		defer release()
		latest, observedView, path, raw, e := loadContractV2(w, *current, id)
		if e != nil {
			return e
		}
		if latest.Local.Pending != nil || latest.Local.AcceptanceReceipt != nil || signing.Digest(latest.Issued.Authority.Payload) != signing.Digest(file.Issued.Authority.Payload) {
			return fmt.Errorf("observed authority/final-return state changed")
		}
		latest.Issued.Authority = document
		updated, e := latest.VerifyIssued(*current)
		if e != nil {
			return e
		}
		if updated.Authority.ContractVersion != state.Contract.Version || updated.Authority.LeaseVersion != state.Contract.LeaseVersion || updated.Authority.ExecutionID != observedView.Authority.ExecutionID || updated.Authority.FencingToken != observedView.Authority.FencingToken || updated.Authority.SpecDigest != observedView.Authority.SpecDigest {
			return fmt.Errorf("refresh cannot take over execution")
		}
		if kind == "review" && observedCaseVersion != latest.Local.ReviewCaseVersion {
			return fmt.Errorf("observed review case CAS changed")
		}
		return w.WriteV2(path, latest, signing.Digest(raw))
	})
	result.ContractID = id
	result.Data = map[string]any{"status": state.Contract, "refreshed": e == nil}
	return result, e
}
