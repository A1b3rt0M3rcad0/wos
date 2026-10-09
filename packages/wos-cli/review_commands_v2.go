package woscli

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"strconv"
)

func checkoutReviewV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: "review checkout"}
	scope, e := scopeV2(w, profile, o)
	if e != nil {
		return result, e
	}
	result.OutcomeID = scope.OutcomeID
	count := 1
	if value := o.values["count"]; value != "" {
		count, e = strconv.Atoi(value)
		if e != nil || count < 1 || count > 10 {
			return result, usage("--count must be 1..10")
		}
	}
	ttl := profile.Lease.RequestedTTLSeconds
	if value := o.values["ttl"]; value != "" {
		ttl, e = strconv.Atoi(value)
		if e != nil || ttl < 30 || ttl > 3600 {
			return result, usage("TTL must be 30..3600 seconds")
		}
	}
	operation := "AcquireSignedReviewContract"
	var command any
	if o.values["next"] == "true" {
		if len(o.args) != 2 {
			return result, usage("review checkout --next --count N")
		}
		limit := 25
		if value := o.values["limit"]; value != "" {
			limit, e = strconv.Atoi(value)
			if e != nil || limit < 1 || limit > 100 {
				return result, usage("scan limit must be 1..100")
			}
		}
		operation = "AcquireNextSignedReviewContract"
		command = a.AcquireNextSignedReviewContractCommand{Scope: scope, SignerKeyID: profile.Signing.KeyID, TTLSeconds: ttl, Limit: limit, Cursor: o.values["cursor"]}
	} else {
		if count != 1 || len(o.args) != 3 {
			return result, usage("review checkout <case-id> --version V or --next --count N")
		}
		id, e := d.ParseID(o.args[2])
		if e != nil {
			return result, e
		}
		version, e := strconv.ParseUint(o.values["version"], 10, 64)
		if e != nil || version == 0 {
			return result, usage("exact review case --version required")
		}
		command = a.AcquireSignedReviewContractCommand{Scope: scope, ReviewCaseID: id, ExpectedReviewCaseVersion: d.Version(version), SignerKeyID: profile.Signing.KeyID, TTLSeconds: ttl}
	}
	intents, e := prepareAcquisitionBatchV2(ctx, w, profile, token, operation, scope, identity.Actor, command, count)
	if e != nil {
		return result, e
	}
	items := []any{}
	acquired := 0
	for _, intent := range intents {
		accepted, e := recoverAcquisitionV2(ctx, w, profile, client, token, intent)
		if !accepted.CommandID.IsZero() {
			result.Committed = true
		}
		if accepted.Acquired {
			acquired++
		}
		items = append(items, map[string]any{"intention_id": intent.ID, "result": compactAcquisitionV2(profile, accepted)})
		result.Data = map[string]any{"requested": count, "acquired": acquired, "partial": acquired < count, "items": items}
		if e != nil {
			result.RequiresAction = "review recover replays original intentions; one active review per Principal remains enforced"
			return result, e
		}
	}
	return result, nil
}

// Public instructions/material remain visible; authentication proofs stay in
// the contract document and are verified by the host, outside model context.
func reviewAgentSpecificationV2(profile ProfileV2, view ContractViewV2) (any, error) {
	spec := view.ReviewSpecification.Spec
	receiptRaw, e := verifyProfileDocumentV2(profile, spec.AcceptanceReceipt, signing.AcceptanceReceipt)
	if e != nil {
		return nil, e
	}
	var receipt signing.ReceiptPayload
	if e = signing.DecodeStrict(receiptRaw, &receipt, signing.MaxPayloadBytes); e != nil {
		return nil, e
	}
	if receipt.SubmissionID != spec.SubmissionID.String() || receipt.SubmissionDigest != spec.SubmissionDigest || receipt.ReviewCaseID != spec.ReviewCaseID.String() {
		return nil, fmt.Errorf("review receipt target differs")
	}
	result := map[string]any{"review_case_id": spec.ReviewCaseID, "submission_id": spec.SubmissionID, "submission_digest": spec.SubmissionDigest, "round": spec.Round, "work_item_version": spec.WorkItemVersion, "execution_principals": spec.ExecutionPrincipals, "execution_groups": spec.ExecutionGroups, "acceptance_floor": spec.AcceptanceFloor, "submission": spec.Submission, "omitted": spec.Omitted, "issued_work_spec_digest": spec.IssuedWorkSpecDigest, "delivery_receipt": receipt}
	if spec.IssuedWorkSpecification != nil {
		raw, e := verifyProfileDocumentV2(profile, *spec.IssuedWorkSpecification, signing.ContractSpec)
		if e != nil {
			return nil, e
		}
		if signing.Digest(raw) != spec.IssuedWorkSpecDigest {
			return nil, fmt.Errorf("original work specification digest differs")
		}
		var work signing.SpecPayload[d.SignedWorkSpec]
		if e = signing.DecodeStrict(raw, &work, 128<<10); e != nil {
			return nil, e
		}
		result["work_specification"] = work.Spec
	}
	return result, nil
}
