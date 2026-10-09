package woscli

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"slices"
)

// This MAC authenticates local recovery provenance, not server authorization.
// It prevents a copied pending file from becoming an intention of another CID.
func frozenMACV2(profile ProfileV2, token []byte, frozen FrozenReturnV2) (string, error) {
	if len(token) < 32 {
		return "", fmt.Errorf("credential unavailable")
	}
	frozen.PendingMAC = ""
	raw, e := json.Marshal(struct {
		Binding string         `json:"binding"`
		Frozen  FrozenReturnV2 `json:"frozen"`
	}{profile.Local.BindingMAC, frozen})
	if e != nil {
		return "", e
	}
	mac := hmac.New(sha256.New, token)
	mac.Write([]byte("wos.local.frozen-return.v2\x00"))
	mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
func sealFrozenV2(profile ProfileV2, token []byte, frozen *FrozenReturnV2) error {
	mac, e := frozenMACV2(profile, token, *frozen)
	if e == nil {
		frozen.PendingMAC = mac
	}
	return e
}
func verifyFrozenV2(profile ProfileV2, token []byte, frozen FrozenReturnV2) (signing.RequestBinding, error) {
	var binding signing.RequestBinding
	if frozen.CredentialID != profile.Binding.CredentialID || !d.ValidSignedDigest(frozen.DraftDigest) || !d.ValidSignedDigest(frozen.RequestDigest) {
		return binding, fmt.Errorf("frozen return provenance differs")
	}
	mac, e := frozenMACV2(profile, token, frozen)
	expected, decode := hex.DecodeString(frozen.PendingMAC)
	actual, _ := hex.DecodeString(mac)
	if e != nil || decode != nil || !hmac.Equal(expected, actual) {
		return binding, fmt.Errorf("frozen return local authentication differs")
	}
	public, e := base64.StdEncoding.Strict().DecodeString(frozen.SignerPublicKey)
	if e != nil || signing.Digest(public) != profile.Signing.PublicKeyFingerprint {
		return binding, fmt.Errorf("frozen signer public key differs")
	}
	raw, e := signing.Verify(frozen.Envelope, frozen.Envelope.PayloadType, profile.Signing.KeyID.String(), ed25519.PublicKey(public))
	if e != nil || len(raw) > 180<<10 || signing.Digest(raw) != frozen.RequestDigest {
		return binding, fmt.Errorf("frozen return payload differs")
	}
	// Strictly decode the entire purpose-specific DTO, then bind the common part.
	switch frozen.Envelope.PayloadType {
	case signing.WorkReturn:
		var request signing.WorkReturnPayload[a.SignedReturnMaterial]
		e = signing.DecodeStrict(raw, &request, 180<<10)
		binding = request.RequestBinding
	case signing.ReviewReturn:
		var request signing.ReviewReturnPayload[a.SignedReviewMaterial]
		e = signing.DecodeStrict(raw, &request, 180<<10)
		binding = request.RequestBinding
	default:
		return binding, fmt.Errorf("frozen return purpose differs")
	}
	if e != nil {
		return binding, e
	}
	if binding.ProtocolVersion != 2 || binding.ServerID != profile.Binding.ServerID.String() || binding.NamespaceID != profile.Binding.NamespaceID.String() || binding.PrincipalID != profile.Binding.PrincipalID || binding.SignerKeyID != profile.Signing.KeyID.String() || d.ID(binding.RequestID).Validate() != nil || d.ID(binding.ContractID).Validate() != nil || d.ID(binding.OutcomeID).Validate() != nil || d.ValidateIdempotencyKey(binding.IdempotencyKey) != nil {
		return binding, fmt.Errorf("frozen return targets differ")
	}
	switch frozen.State {
	case "prepared_signed", "sent_unknown", "accepted_confirmed", "cleanup_pending":
	default:
		return binding, fmt.Errorf("frozen return state differs")
	}
	return binding, nil
}
func returnOperationV2(kind string) string {
	if kind == "review" {
		return "ReturnSignedReview"
	}
	return "ReturnSignedWork"
}
func returnIntentV2(profile ProfileV2, frozen FrozenReturnV2, b signing.RequestBinding, actor d.ActorRef) (PendingOperationV2, error) {
	raw, e := json.Marshal(frozen)
	if e != nil || len(raw) > 256<<10 {
		return PendingOperationV2{}, fmt.Errorf("frozen return technical limit exceeded")
	}
	var command any = a.ReturnSignedWorkCommand{Envelope: frozen.Envelope}
	if frozen.Envelope.PayloadType == signing.ReviewReturn {
		command = a.ReturnSignedReviewCommand{Envelope: frozen.Envelope}
	}
	fp, e := a.SignedCommandFingerprint(actor, command)
	if e != nil {
		return PendingOperationV2{}, e
	}
	id := d.ID(b.ContractID)
	// Payload never changes after preparation. Response/state live separately.
	return PendingOperationV2{ID: d.ID(b.RequestID), Operation: returnOperationV2(func() string {
		if frozen.Envelope.PayloadType == signing.ReviewReturn {
			return "review"
		}
		return "execution"
	}()), State: "prepared_signed", Scope: ProfilePendingScopeV2{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(b.OutcomeID)}, ContractID: &id, IdempotencyKey: b.IdempotencyKey, Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw), RequestFingerprint: fp}, nil
}
func liveReturnPreflightV2(profile ProfileV2, file ContractFileV2, view ContractViewV2, identity a.SigningIdentityView, state a.SignedStateResult, grant a.SignedStateResult, review a.SignedStateResult) error {
	c := state.Contract
	g := view.Authority
	if c == nil || c.CredentialID != profile.Binding.CredentialID || !c.Current || !c.LeaseValid || c.ID.String() != g.ContractID || c.ExecutionID.String() != g.ExecutionID || c.FencingToken != g.FencingToken || c.Version != g.ContractVersion || c.LeaseVersion != g.LeaseVersion || c.WorkItemVersion != file.Local.WorkItemVersion || c.SpecDigest != g.SpecDigest {
		return fmt.Errorf("live authority/CID/CAS differs; refresh explicitly without rebasing the draft")
	}
	if grant.Envelope == nil {
		return fmt.Errorf("current authority proof unavailable")
	}
	document, e := signing.ToDocument(*grant.Envelope)
	if e != nil {
		return e
	}
	raw, e := verifyProfileDocumentV2(profile, document, signing.ContractAuthority)
	if e != nil || signing.Digest(raw) != signing.Digest(file.Issued.Authority.Payload) {
		return fmt.Errorf("current authority proof differs")
	}
	p := identity.Policy
	if p == nil || p.CredentialID != profile.Binding.CredentialID || p.PrincipalID != profile.Binding.PrincipalID || p.Version == 0 || !slices.Contains(p.AllowedSigningKeyIDs, profile.Signing.KeyID) || len(p.AllowedOutcomeIDs) > 0 && !slices.Contains(p.AllowedOutcomeIDs, d.ID(g.OutcomeID)) || !slices.Contains(g.AllowedSigningKeyIDs, profile.Signing.KeyID.String()) {
		return fmt.Errorf("current signing policy does not authorize this key/scope")
	}
	permission := "work.contract.return"
	if g.ContractKind == "review" {
		permission = "work.review.decide"
	}
	if !slices.Contains(p.PermittedOperations, permission) {
		return fmt.Errorf("current policy does not permit this return")
	}
	active := false
	for _, key := range identity.Keys {
		if key.ID == profile.Signing.KeyID && key.PrincipalID == profile.Binding.PrincipalID && key.NamespaceID == profile.Binding.NamespaceID && key.Status == "active" && key.Fingerprint == profile.Signing.PublicKeyFingerprint {
			active = true
		}
	}
	if !active {
		return fmt.Errorf("selected own signing key is not active or is outside the bounded identity view")
	}
	if g.ContractKind == "review" && (review.ReviewCase == nil || signing.Decimal(review.ReviewCase.Version) != file.Local.ReviewCaseVersion || review.ReviewCase.ID.String() != g.ReviewCaseID) {
		return fmt.Errorf("review case CAS differs; preserve draft")
	}
	return nil
}
func prepareReturnV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, id d.ID, kind, completion string) (PendingOperationV2, error) {
	var intent PendingOperationV2
	file, view, _, observed, e := loadContractV2(w, profile, id)
	if e != nil {
		return intent, e
	}
	if view.Authority.ContractKind != kind {
		return intent, fmt.Errorf("contract kind differs")
	}
	if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
		return intent, fmt.Errorf("return already frozen or accepted; send/recover its original intention")
	}
	if completion != "" {
		if kind != "execution" || completion != file.Execution.CompletionIntent {
			return intent, fmt.Errorf("--completion must match the editable draft; edit it explicitly before signing")
		}
	}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: kind, ID: id})
	if e != nil {
		return intent, e
	}
	grant, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "authority", ID: id, ContractKind: kind})
	if e != nil {
		return intent, e
	}
	var review a.SignedStateResult
	if kind == "review" {
		review, e = client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "case", ID: d.ID(view.Authority.ReviewCaseID)})
		if e != nil {
			return intent, e
		}
	}
	if e = liveReturnPreflightV2(profile, file, view, identity, state, grant, review); e != nil {
		return intent, e
	}
	private, e := readProfilePrivateV2(profile, defaultSecretResolverV2(w))
	if e != nil {
		return intent, e
	}
	defer clear(private)
	requestID, e := newLocalIDV2()
	if e != nil {
		return intent, e
	}
	key, e := sdk.NewIdempotencyKey()
	if e != nil {
		return intent, e
	}
	g := view.Authority
	binding := signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: g.ServerID, NamespaceID: g.NamespaceID, OutcomeID: g.OutcomeID, PrincipalID: g.PrincipalID, SignerKeyID: profile.Signing.KeyID.String()}, OperationKind: "work_return", RequestID: requestID.String(), IdempotencyKey: key, ContractID: g.ContractID, WorkItemID: g.WorkItemID, ExecutionID: g.ExecutionID, FencingToken: g.FencingToken, SpecDigest: g.SpecDigest, AuthorityDigest: signing.Digest(file.Issued.Authority.Payload), ExpectedContractVersion: g.ContractVersion, ExpectedLeaseVersion: g.LeaseVersion, ExpectedWorkItemVersion: file.Local.WorkItemVersion, PolicyRevision: signing.Decimal(identity.Policy.Version)}
	var payload any
	purpose := signing.WorkReturn
	if kind == "execution" {
		switch file.Execution.CompletionIntent {
		case "auto", "direct", "review":
		default:
			return intent, fmt.Errorf("completion intent must be auto/direct/review")
		}
		request := signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: binding, CompletionIntent: file.Execution.CompletionIntent, Material: file.Execution.Material}
		if previous := view.WorkSpecification.Spec.PreviousSubmissionID; previous != nil {
			request.SupersedesSubmissionID = previous.String()
		}
		payload = request
	} else {
		binding.OperationKind = "review_return"
		purpose = signing.ReviewReturn
		switch file.Review.Decision {
		case "approved", "changes_requested", "inconclusive":
		default:
			return intent, fmt.Errorf("review decision differs")
		}
		spec := view.ReviewSpecification.Spec
		payload = signing.ReviewReturnPayload[a.SignedReviewMaterial]{RequestBinding: binding, ExpectedReviewCaseVersion: file.Local.ReviewCaseVersion, ReviewCaseID: spec.ReviewCaseID.String(), SubmissionID: spec.SubmissionID.String(), SubmissionDigest: spec.SubmissionDigest, Decision: file.Review.Decision, Material: file.Review.Material}
	}
	raw, e := signing.Canonical(payload)
	if e != nil {
		return intent, e
	}
	if len(raw) > 180<<10 {
		return intent, fmt.Errorf("return payload exceeds 180 KiB")
	}
	envelope, e := signing.SignCanonical(purpose, raw, profile.Signing.KeyID.String(), ed25519.PrivateKey(private))
	if e != nil {
		return intent, e
	}
	draft, e := file.DraftDigest()
	if e != nil {
		return intent, e
	}
	frozen := FrozenReturnV2{State: "prepared_signed", Envelope: envelope, DraftDigest: draft, RequestDigest: signing.Digest(raw), CredentialID: profile.Binding.CredentialID, SignerPublicKey: base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey))}
	if e = sealFrozenV2(profile, token, &frozen); e != nil {
		return intent, e
	}
	intent, e = returnIntentV2(profile, frozen, binding, identity.Actor)
	if e != nil {
		return intent, e
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		if len(current.Local.PendingOperations) >= 10 {
			return fmt.Errorf("pending limit reached")
		}
		for _, pending := range current.Local.PendingOperations {
			if pending.ContractID != nil && *pending.ContractID == id {
				return fmt.Errorf("contract has another unresolved intention")
			}
		}
		unlock, e := w.LockV2(ctx, current.Name, id.String())
		if e != nil {
			return e
		}
		defer unlock()
		latest, _, path, bytes, e := loadContractV2(w, *current, id)
		if e != nil {
			return e
		}
		if signing.Digest(bytes) != signing.Digest(observed) {
			return fmt.Errorf("draft changed during preflight; no signature persisted")
		}
		latest.Local.Pending = &frozen
		if e = w.WriteV2(path, latest, signing.Digest(bytes)); e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
		return nil
	})
	return intent, e
}
