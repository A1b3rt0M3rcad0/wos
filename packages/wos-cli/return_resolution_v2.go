package woscli

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// A separate journal for the new intention. The parent frozen payload and its
// signature are never rewritten. The profile pending MAC authenticates both.
type PendingReturnResolutionV2 struct {
	State         string           `json:"state"`
	Envelope      signing.Envelope `json:"envelope"`
	RequestDigest string           `json:"request_digest"`
	Response      string           `json:"response,omitempty"`
}

func validatePendingResolutionV2(profile ProfileV2, intent PendingOperationV2) error {
	if intent.Resolution == nil {
		return nil
	}
	r := intent.Resolution
	if !returnPendingV2(intent.Operation) || r.Envelope.PayloadType != signing.ReturnResolution || !d.ValidSignedDigest(r.RequestDigest) {
		return fmt.Errorf("invalid return resolution journal")
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(r.Envelope.Payload)
	if e != nil || len(raw) > 32<<10 || signing.Digest(raw) != r.RequestDigest {
		return fmt.Errorf("resolution payload bytes differ")
	}
	if r.Response != "" && !validTechnicalV2Blob(r.Response) {
		return fmt.Errorf("invalid resolution response")
	}
	switch r.State {
	case "resolution_prepared", "resolution_unknown", "abandoned_confirmed", "accepted_confirmed", "cleanup_pending":
	default:
		return fmt.Errorf("invalid return resolution state")
	}
	return nil
}

func decodeResolutionV2(profile ProfileV2, token []byte, intent PendingOperationV2) (FrozenReturnV2, signing.RequestBinding, signing.ReturnResolutionPayload, error) {
	frozen, b, e := decodeReturnIntentV2(profile, token, intent)
	var request signing.ReturnResolutionPayload
	if e != nil {
		return frozen, b, request, e
	}
	if intent.Resolution == nil {
		return frozen, b, request, fmt.Errorf("resolution has not been explicitly prepared")
	}
	if e = validatePendingResolutionV2(profile, intent); e != nil {
		return frozen, b, request, e
	}
	pub, e := base64.StdEncoding.Strict().DecodeString(frozen.SignerPublicKey)
	if e != nil {
		return frozen, b, request, e
	}
	raw, e := signing.Verify(intent.Resolution.Envelope, signing.ReturnResolution, profile.Signing.KeyID.String(), ed25519.PublicKey(pub))
	if e != nil {
		return frozen, b, request, e
	}
	if e = signing.DecodeStrict(raw, &request, 32<<10); e != nil {
		return frozen, b, request, e
	}
	envBytes, e := signing.Canonical(frozen.Envelope)
	if e != nil {
		return frozen, b, request, e
	}
	if request.ProtocolVersion != 2 || request.OperationKind != "resolve_return_intent" || request.ServerID != b.ServerID || request.NamespaceID != b.NamespaceID || request.OutcomeID != b.OutcomeID || request.PrincipalID != b.PrincipalID || request.SignerKeyID != b.SignerKeyID || request.CredentialID != frozen.CredentialID.String() || request.ContractID != b.ContractID || request.OriginalCommandName != intent.Operation || request.OriginalRequestID != b.RequestID || request.OriginalIdempotencyKey != b.IdempotencyKey || request.OriginalRequestDigest != frozen.RequestDigest || request.OriginalEnvelopeDigest != signing.Digest(envBytes) || request.RequestID == b.RequestID || request.IdempotencyKey == b.IdempotencyKey || d.ID(request.RequestID).Validate() != nil || d.ValidateIdempotencyKey(request.IdempotencyKey) != nil || strings.TrimSpace(request.Reason) == "" || len(request.Reason) > 4096 || request.PolicyRevision == 0 {
		return frozen, b, request, fmt.Errorf("resolution differs from the exact original frozen intention")
	}
	return frozen, b, request, nil
}

func verifyResolutionReceiptV2(profile ProfileV2, token []byte, intent PendingOperationV2, doc signing.Document) error {
	_, _, request, e := decodeResolutionV2(profile, token, intent)
	if e != nil {
		return e
	}
	raw, e := verifyProfileDocumentV2(profile, doc, signing.ReturnResolutionReceipt)
	if e != nil {
		return e
	}
	var proof signing.ReturnResolutionReceiptPayload
	if e = signing.DecodeStrict(raw, &proof, 32<<10); e != nil {
		return e
	}
	if proof.ProtocolVersion != 2 || proof.ServerID != request.ServerID || proof.NamespaceID != request.NamespaceID || proof.OutcomeID != request.OutcomeID || proof.PrincipalID != request.PrincipalID || proof.CredentialID != request.CredentialID || proof.ContractID != request.ContractID || proof.RequestID != request.RequestID || proof.IdempotencyKey != request.IdempotencyKey || proof.RequestDigest != intent.Resolution.RequestDigest || proof.OriginalCommandName != request.OriginalCommandName || proof.OriginalRequestID != request.OriginalRequestID || proof.OriginalIdempotencyKey != request.OriginalIdempotencyKey || proof.OriginalRequestDigest != request.OriginalRequestDigest || proof.OriginalEnvelopeDigest != request.OriginalEnvelopeDigest || proof.Decision != "closed_without_acceptance" || proof.Reason != request.Reason || proof.OutcomeRevision == 0 || proof.ResolvedAt == "" {
		return fmt.Errorf("terminal receipt does not close this exact original intention")
	}
	return nil
}

func prepareResolutionV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, identity a.SigningIdentityView, reason string, private ed25519.PrivateKey) (PendingOperationV2, error) {
	if strings.TrimSpace(reason) == "" || len(reason) > 4096 {
		return intent, usage("resolve requires an explicit --reason of at most 4096 bytes")
	}
	if identity.Policy == nil || identity.CredentialID != profile.Binding.CredentialID || identity.PrincipalID != profile.Binding.PrincipalID {
		return intent, fmt.Errorf("current original credential policy required")
	}
	frozen, b, e := decodeReturnIntentV2(profile, token, intent)
	if e != nil {
		return intent, e
	}
	if intent.State == "accepted_confirmed" || intent.State == "cleanup_pending" {
		return intent, fmt.Errorf("accepted return must finish normal receipt cleanup")
	}
	if intent.Resolution != nil {
		return intent, fmt.Errorf("resolution already prepared; recover the same intention")
	}
	envBytes, e := signing.Canonical(frozen.Envelope)
	if e != nil {
		return intent, e
	}
	id, e := newLocalIDV2()
	if e != nil {
		return intent, e
	}
	key, e := sdk.NewIdempotencyKey()
	if e != nil {
		return intent, e
	}
	p := signing.ReturnResolutionPayload{Binding: b.Binding, OperationKind: "resolve_return_intent", RequestID: id.String(), IdempotencyKey: key, CredentialID: profile.Binding.CredentialID.String(), OriginalCommandName: intent.Operation, OriginalRequestID: b.RequestID, OriginalIdempotencyKey: b.IdempotencyKey, OriginalRequestDigest: frozen.RequestDigest, OriginalEnvelopeDigest: signing.Digest(envBytes), ContractID: b.ContractID, PolicyRevision: signing.Decimal(identity.Policy.Version), Reason: reason}
	env, e := signing.Sign(signing.ReturnResolution, p, profile.Signing.KeyID.String(), private)
	if e != nil {
		return intent, e
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(env.Payload)
	if e != nil {
		return intent, e
	}
	intent.Resolution = &PendingReturnResolutionV2{State: "resolution_prepared", Envelope: env, RequestDigest: signing.Digest(raw)}
	if _, _, _, e = decodeResolutionV2(profile, token, intent); e != nil {
		return intent, e
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		if current.Local.PendingOperations[index].Resolution != nil {
			return fmt.Errorf("another resolution already exists; preserve its signature")
		}
		current.Local.PendingOperations[index].Resolution = intent.Resolution
		return nil
	})
	return intent, e
}

func updateResolutionStateV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, state string, doc *signing.Document) error {
	return mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		pending := &current.Local.PendingOperations[index]
		if pending.Resolution == nil || intent.Resolution == nil || pending.Resolution.RequestDigest != intent.Resolution.RequestDigest {
			return fmt.Errorf("resolution journal changed")
		}
		if doc != nil {
			if state == "accepted_confirmed" {
				frozen, b, err := decodeReturnIntentV2(*current, token, *pending)
				if err != nil {
					return err
				}
				if _, err = verifyReturnReceiptV2(*current, frozen, b, *doc); err != nil {
					return err
				}
			} else if e = verifyResolutionReceiptV2(*current, token, *pending, *doc); e != nil {
				return e
			}
			raw, e := json.Marshal(doc)
			if e != nil {
				return e
			}
			pending.Resolution.Response = base64.StdEncoding.EncodeToString(raw)
		}
		pending.Resolution.State = state
		return nil
	})
}

type returnResolutionArchiveV2 struct {
	SchemaVersion  int                `json:"schema_version"`
	Kind           string             `json:"kind"`
	OriginalIntent PendingOperationV2 `json:"original_intent"`
	ContractPath   string             `json:"contract_path"`
	ContractDigest string             `json:"contract_digest"`
	Receipt        signing.Document   `json:"resolution_receipt"`
}

func cleanupResolutionV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, doc signing.Document, hook func(string) error) error {
	return mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		pending := current.Local.PendingOperations[index]
		frozen, b, _, e := decodeResolutionV2(*current, token, pending)
		if e != nil {
			return e
		}
		if e = verifyResolutionReceiptV2(*current, token, pending, doc); e != nil {
			return e
		}
		if pending.Resolution.State != "abandoned_confirmed" && pending.Resolution.State != "cleanup_pending" {
			return fmt.Errorf("terminal proof must be persisted before cleanup")
		}
		proofBytes, e := base64.StdEncoding.Strict().DecodeString(pending.Resolution.Response)
		if e != nil {
			return e
		}
		var persisted signing.Document
		if e = signing.DecodeStrict(proofBytes, &persisted, 32<<10); e != nil {
			return e
		}
		if e = verifyResolutionReceiptV2(*current, token, pending, persisted); e != nil {
			return e
		}
		if signing.Digest(persisted.Payload) != signing.Digest(doc.Payload) {
			return fmt.Errorf("persisted terminal proof differs")
		}
		unlock, e := w.LockV2(ctx, current.Name, b.ContractID)
		if e != nil {
			return e
		}
		defer unlock()
		file, _, path, observed, e := loadContractV2(w, *current, *intent.ContractID)
		if e != nil {
			return e
		}
		archivePath := filepath.Join(".wos/profiles", current.Name, "history", "return-"+b.RequestID+".yaml")
		draftPath := filepath.Join(".wos/profiles", current.Name, "history", "return-"+b.RequestID+".contract.yaml")
		savedDraft, draftErr := w.ReadV2(draftPath)
		if errors.Is(draftErr, os.ErrNotExist) {
			if file.Local.Pending == nil || file.Local.Pending.RequestDigest != frozen.RequestDigest {
				return fmt.Errorf("original local frozen return missing before archive")
			}
			if _, e = verifyFrozenV2(*current, token, *file.Local.Pending); e != nil {
				return e
			}
			if e = w.writeRawV2(draftPath, observed, "", true); e != nil {
				return e
			}
			savedDraft = observed
			if hook != nil {
				if e = hook("draft_archived"); e != nil {
					return e
				}
			}
		} else if draftErr != nil {
			return draftErr
		}
		var archivedFile ContractFileV2
		if e = DecodeV2Document(savedDraft, &archivedFile); e != nil {
			return e
		}
		if archivedFile.Local.Pending == nil || archivedFile.Local.Pending.RequestDigest != frozen.RequestDigest {
			return fmt.Errorf("archived draft does not preserve the original frozen return")
		}
		if _, e = verifyFrozenV2(*current, token, *archivedFile.Local.Pending); e != nil {
			return e
		}
		archiveRaw, archiveErr := w.ReadV2(archivePath)
		if errors.Is(archiveErr, os.ErrNotExist) {
			archive := returnResolutionArchiveV2{1, "WOSReturnResolutionArchive", pending, draftPath, signing.Digest(savedDraft), doc}
			if e = w.CreateV2(archivePath, archive); e != nil {
				return e
			}
			if hook != nil {
				if e = hook("archive_persisted"); e != nil {
					return e
				}
			}
		} else {
			if archiveErr != nil {
				return archiveErr
			}
			var archive returnResolutionArchiveV2
			if e = DecodeV2Document(archiveRaw, &archive); e != nil {
				return e
			}
			if archive.ContractPath != draftPath || signing.Digest(savedDraft) != archive.ContractDigest {
				return fmt.Errorf("archived draft bytes differ")
			}
			if archive.SchemaVersion != 1 || archive.Kind != "WOSReturnResolutionArchive" || archive.OriginalIntent.ID != pending.ID || archive.OriginalIntent.Payload != pending.Payload || archive.OriginalIntent.PayloadDigest != pending.PayloadDigest {
				return fmt.Errorf("archive belongs to a different original intention")
			}
			if e = verifyResolutionReceiptV2(*current, token, pending, archive.Receipt); e != nil {
				return e
			}
		}
		if file.Local.Pending != nil {
			if file.Local.Pending.RequestDigest != frozen.RequestDigest {
				return fmt.Errorf("another frozen return is present; preserve file")
			}
			if _, e = verifyFrozenV2(*current, token, *file.Local.Pending); e != nil {
				return e
			}
			// Preserve the entire current editable draft, including edits made after
			// signing. Only its proven-terminal local pending marker is cleared.
			file.Local.Pending = nil
			if e = w.WriteV2(path, file, signing.Digest(observed)); e != nil {
				return e
			}
			if hook != nil {
				if e = hook("draft_unfrozen"); e != nil {
					return e
				}
			}
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
}

func recoverResolutionV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2) (signing.Document, bool, error) {
	var doc signing.Document
	frozen, b, request, e := decodeResolutionV2(profile, token, intent)
	if e != nil {
		return doc, false, e
	}
	if intent.Resolution.State == "accepted_confirmed" {
		raw, e := base64.StdEncoding.Strict().DecodeString(intent.Resolution.Response)
		if e != nil {
			return doc, false, e
		}
		if e = signing.DecodeStrict(raw, &doc, 256<<10); e != nil {
			return doc, false, e
		}
		if _, e = verifyReturnReceiptV2(profile, frozen, b, doc); e != nil {
			return doc, false, e
		}
		if e = updateReturnStateV2(ctx, w, profile, token, intent, "accepted_confirmed", &doc); e != nil {
			return doc, true, e
		}
		return doc, true, cleanupReturnV2(ctx, w, profile, token, intent, doc)
	}
	if intent.Resolution.State == "abandoned_confirmed" || intent.Resolution.State == "cleanup_pending" {
		raw, e := base64.StdEncoding.Strict().DecodeString(intent.Resolution.Response)
		if e != nil {
			return doc, false, e
		}
		if e = signing.DecodeStrict(raw, &doc, 32<<10); e != nil {
			return doc, false, e
		}
		return doc, true, cleanupResolutionV2(ctx, w, profile, token, intent, doc, nil)
	}
	if e = updateResolutionStateV2(ctx, w, profile, token, intent, "resolution_unknown", nil); e != nil {
		return doc, false, e
	}
	// No negative inference from either lookup. Missing proof leads only to an
	// exact retry of the previously frozen resolution, never a new intention.
	doc, e = lookupReturnReceiptV2(ctx, client, profile, intent, frozen, b)
	accepted := e == nil
	if e != nil && !notFoundReturnV2(e) {
		return doc, false, e
	}
	if !accepted {
		kind := "execution"
		if intent.Operation == "ReturnSignedReview" {
			kind = "review"
		}
		state, err := client.ReadSignedReturnResolution(ctx, d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}, kind, intent.IdempotencyKey)
		if err == nil && state.Envelope != nil {
			doc, err = signing.ToDocument(*state.Envelope)
			if err == nil {
				err = verifyResolutionReceiptV2(profile, token, intent, doc)
			}
		} else if err == nil {
			err = fmt.Errorf("terminal proof missing")
		}
		if err != nil && !notFoundReturnV2(err) {
			return doc, false, err
		}
		if err != nil {
			result, err := client.ResolveSignedReturnIntent(ctx, request.IdempotencyKey, a.ResolveSignedReturnIntentCommand{Scope: d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}, OriginalEnvelope: frozen.Envelope, Envelope: intent.Resolution.Envelope})
			if err != nil {
				return doc, false, err
			}
			if result.ResultOmitted {
				return doc, false, fmt.Errorf("resolution result omitted; recover the exact signed proof")
			}
			switch result.Value.Decision {
			case "accepted":
				if result.Value.AcceptanceReceipt == nil || result.Value.ResolutionReceipt != nil {
					return doc, false, fmt.Errorf("acceptance proof missing")
				}
				doc = *result.Value.AcceptanceReceipt
				accepted = true
			case "closed_without_acceptance":
				if result.Value.ResolutionReceipt == nil || result.Value.AcceptanceReceipt != nil {
					return doc, false, fmt.Errorf("terminal proof missing")
				}
				doc = *result.Value.ResolutionReceipt
			default:
				return doc, false, fmt.Errorf("unknown authoritative resolution decision")
			}
		}
	}
	if accepted {
		if _, e = verifyReturnReceiptV2(profile, frozen, b, doc); e != nil {
			return doc, false, e
		}
		if e = updateResolutionStateV2(ctx, w, profile, token, intent, "accepted_confirmed", &doc); e != nil {
			return doc, true, e
		}
		if e = updateReturnStateV2(ctx, w, profile, token, intent, "accepted_confirmed", &doc); e != nil {
			return doc, true, e
		}
		return doc, true, cleanupReturnV2(ctx, w, profile, token, intent, doc)
	}
	if e = verifyResolutionReceiptV2(profile, token, intent, doc); e != nil {
		return doc, false, e
	}
	if e = updateResolutionStateV2(ctx, w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
		return doc, true, e
	}
	return doc, true, cleanupResolutionV2(ctx, w, profile, token, intent, doc, nil)
}

func returnResolutionCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: o.args[0] + " " + o.args[1]}
	if len(o.args) != 3 {
		return result, usage("resolve/resolution-status requires one contract ID")
	}
	id, e := d.ParseID(o.args[2])
	if e != nil {
		return result, e
	}
	result.ContractID = id
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		return result, e
	}
	if e = current.VerifyBinding(token); e != nil {
		return result, e
	}
	var intent PendingOperationV2
	for _, p := range current.Local.PendingOperations {
		if p.ContractID != nil && *p.ContractID == id && returnPendingV2(p.Operation) {
			intent = p
			break
		}
	}
	if intent.ID.IsZero() {
		return result, fmt.Errorf("no frozen return journal for this contract; preserve local files")
	}
	if (o.args[0] == "review") != (intent.Operation == "ReturnSignedReview") {
		return result, usage("return kind differs")
	}
	if o.args[1] == "resolution-status" {
		state := intent.State
		var resolutionID string
		if intent.Resolution != nil {
			_, _, p, err := decodeResolutionV2(current, token, intent)
			if err != nil {
				return result, err
			}
			state = intent.Resolution.State
			resolutionID = p.RequestID
		}
		confirmed := false
		if intent.Resolution != nil && (state == "accepted_confirmed" || state == "abandoned_confirmed" || state == "cleanup_pending") {
			raw, err := base64.StdEncoding.Strict().DecodeString(intent.Resolution.Response)
			if err != nil {
				return result, err
			}
			var proof signing.Document
			if err = signing.DecodeStrict(raw, &proof, 256<<10); err != nil {
				return result, err
			}
			if state == "accepted_confirmed" {
				frozen, b, err := decodeReturnIntentV2(current, token, intent)
				if err != nil {
					return result, err
				}
				if _, err = verifyReturnReceiptV2(current, frozen, b, proof); err != nil {
					return result, err
				}
			} else if err = verifyResolutionReceiptV2(current, token, intent, proof); err != nil {
				return result, err
			}
			confirmed = true
		}
		result.Data = map[string]any{"original_request_id": intent.ID, "state": state, "resolution_request_id": resolutionID, "server_decision_confirmed": confirmed, "cleanup_pending": confirmed, "authority": "only a verified server receipt authorizes local cleanup"}
		return result, nil
	}
	if intent.Resolution == nil {
		capabilities, err := client.Capabilities(ctx)
		if err != nil {
			return result, err
		}
		if !capabilities.SignedReturnResolution {
			return result, fmt.Errorf("server does not advertise terminal return resolution; preserve original")
		}
		private, err := readProfilePrivateV2(current, defaultSecretResolverV2(w))
		if err != nil {
			return result, err
		}
		defer clear(private)
		intent, e = prepareResolutionV2(ctx, w, current, token, intent, identity, o.values["reason"], ed25519.PrivateKey(private))
		if e != nil {
			return result, e
		}
	} else if o.values["reason"] != "" {
		_, _, p, err := decodeResolutionV2(current, token, intent)
		if err != nil {
			return result, err
		}
		if p.Reason != o.values["reason"] {
			return result, fmt.Errorf("resolution reason is already frozen; recover it unchanged")
		}
	}
	doc, committed, e := recoverResolutionV2(ctx, w, current, client, token, intent)
	result.Committed = committed
	result.Data = map[string]any{"original_request_id": intent.ID, "server_decision_confirmed": committed, "proof_digest": signing.Digest(doc.Payload), "cleanup_pending": committed && e != nil}
	if committed {
		result.Data.(map[string]any)["original_return_accepted"] = doc.Proof.PayloadType == signing.AcceptanceReceipt
	}
	if e != nil {
		result.RequiresAction = "preserve both frozen intentions; recover the original resolution receipt without changing authority"
	}
	return result, e
}
