package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
)

type SignedFindingResponse struct {
	FindingID         d.ID     `json:"finding_id"`
	Summary           string   `json:"summary"`
	EvidenceIDs       []d.ID   `json:"evidence_ids,omitempty"`
	EvidenceLocalKeys []string `json:"evidence_local_keys,omitempty"`
}

func validateCorrectionResponses(findings []d.SignedFinding, responses []SignedFindingResponse, material d.WorkResultMaterial, local map[string]d.ID) error {
	if len(responses) > 100 || len(responses) != len(findings) {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "correction must explicitly address every frozen finding")
	}
	required := map[d.ID]bool{}
	for _, f := range findings {
		required[f.ID] = true
	}
	evidence := map[d.ID]bool{}
	for _, id := range material.EvidenceIDs {
		evidence[id] = true
	}
	seen := map[d.ID]bool{}
	for _, r := range responses {
		if !required[r.FindingID] || seen[r.FindingID] || strings.TrimSpace(r.Summary) == "" || len(r.Summary) > 16384 {
			return d.NewError(d.ErrorCodeInvalidArgument, "finding response is unknown/duplicate/empty")
		}
		seen[r.FindingID] = true
		ids := append([]d.ID(nil), r.EvidenceIDs...)
		for _, key := range r.EvidenceLocalKeys {
			id, ok := local[key]
			if !ok {
				return d.NewError(d.ErrorCodeInvalidArgument, "unknown finding evidence local key")
			}
			ids = append(ids, id)
		}
		for _, id := range ids {
			if !evidence[id] {
				return d.NewError(d.ErrorCodeSubmissionNotAccepted, "finding evidence is outside returned material")
			}
		}
	}
	return nil
}
func verifiedReviewDecision(ctx context.Context, u ports.UnitOfWork, f d.SignedFact, serverID string) (signing.ReviewReturnPayload[SignedReviewMaterial], error) {
	var zero signing.ReviewReturnPayload[SignedReviewMaterial]
	if f.Kind != "return" || f.ContractKind != "review" {
		return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "signed review decision fact required")
	}
	registry, err := signingRepository(u)
	if err != nil {
		return zero, err
	}
	key, err := registry.Key(ctx, f.Scope.NamespaceID, f.KeyID)
	if err != nil {
		return zero, err
	}
	if key.Purpose != "agent" || key.PrincipalID != f.PrincipalID {
		return zero, d.NewError(d.ErrorCodeForbidden, "review decision historical signer differs")
	}
	doc, err := factDocument(f)
	if err != nil {
		return zero, err
	}
	envelope, err := doc.Envelope()
	if err != nil {
		return zero, signingError(err)
	}
	pub, err := key.PublicBytes()
	if err != nil {
		return zero, err
	}
	raw, err := signing.Verify(envelope, signing.ReviewReturn, key.ID.String(), pub)
	if err != nil {
		return zero, signingError(err)
	}
	if err = signing.DecodeStrict(raw, &zero, 180*1024); err != nil {
		return zero, signingError(err)
	}
	if zero.ServerID != serverID || zero.NamespaceID != f.Scope.NamespaceID.String() || zero.OutcomeID != f.Scope.OutcomeID.String() || zero.PrincipalID != f.PrincipalID || zero.ContractID != f.ContractID.String() || zero.OperationKind != "review_return" || f.RequestID == nil || zero.RequestID != f.RequestID.String() || zero.IdempotencyKey != f.IdempotencyKey || signing.Digest(raw) != f.RequestDigest {
		return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated historical review decision binding differs")
	}
	repo, _, err := signedRepository(u)
	if err != nil {
		return zero, err
	}
	acceptance, err := repo.Acceptance(ctx, f.Scope.NamespaceID, f.PrincipalID, f.IdempotencyKey)
	if err != nil {
		return zero, err
	}
	receiptDoc, err := verifyIssuedFact(ctx, u, acceptance, signing.AcceptanceReceipt, serverID, f.PrincipalID)
	if err != nil {
		return zero, err
	}
	var receipt signing.ReceiptPayload
	if err = signing.DecodeStrict(receiptDoc.Payload, &receipt, 180*1024); err != nil {
		return zero, signingError(err)
	}
	disposition := map[string]string{"approved": "review_approved", "changes_requested": "changes_requested", "inconclusive": "review_inconclusive"}[zero.Decision]
	if !receipt.Accepted || !receipt.LocalObligationClosed || receipt.RequestDigest != f.RequestDigest || receipt.RequestID != zero.RequestID || receipt.IdempotencyKey != f.IdempotencyKey || receipt.DecisionID != f.ID.String() || receipt.ContractID != f.ContractID.String() || receipt.ReviewCaseID != zero.ReviewCaseID || receipt.SubmissionID != zero.SubmissionID || receipt.SubmissionDigest != zero.SubmissionDigest || receipt.Disposition != disposition {
		return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "historical review acceptance does not bind these findings")
	}
	return zero, nil
}
