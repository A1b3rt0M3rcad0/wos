package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func requireActivityCapabilityV2(ctx context.Context, client *sdk.Client, attention bool) error {
	capabilities, e := client.Capabilities(ctx)
	if e != nil {
		return e
	}
	if attention && capabilities.WorkAttentionVersion < 1 || !attention && capabilities.WorkActivityVersion < 1 {
		return usage("this server does not advertise the required work activity/attention capability; upgrade the server; unsigned fallback is forbidden")
	}
	return nil
}

type workTargetV2 struct {
	Scope  d.Scope
	ID     d.ID
	WorkID d.ID
	File   *ContractFileV2
	View   ContractViewV2
	State  a.SignedStateResult
}

func resolveWorkTargetV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, kind string, o options, position int) (workTargetV2, error) {
	var target workTargetV2
	if ref := o.values["work"]; ref != "" {
		if len(o.args) > position || o.values["contract"] != "" {
			return target, usage("--work cannot be combined with a contract reference")
		}
		scope, e := resolveOutcomeV2(ctx, w, profile, client, o)
		if e != nil {
			return target, e
		}
		id, e := d.ParseID(ref)
		if e != nil {
			return target, usage("historical --work queries require the exact WorkItem ID")
		}
		target.Scope, target.WorkID = scope, id
		return target, nil
	}
	id, e := contractOptionV2(w, profile, kind, o, position)
	if e != nil {
		return target, e
	}
	target.ID = id
	file, view, _, _, e := loadContractV2(w, profile, id)
	if e == nil {
		if view.Authority.ContractKind != kind {
			return target, usage("contract kind differs from selected command")
		}
		target.Scope = d.Scope{TenantID: profile.Binding.TenantID, OutcomeID: d.ID(view.Authority.OutcomeID)}
		target.File, target.View = &file, view
	} else {
		// Missing accepted local material is normal after receipt-confirmed
		// cleanup. Malformed or modified local material is never bypassed.
		if !os.IsNotExist(e) {
			return target, e
		}
		target.Scope, e = resolveOutcomeV2(ctx, w, profile, client, o)
		if e != nil {
			return target, e
		}
	}
	target.State, e = client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: kind, ID: id})
	if e != nil {
		return target, e
	}
	if target.State.Contract == nil || target.State.Contract.ID != id || target.State.Contract.Kind != kind {
		return target, fmt.Errorf("server contract query target differs")
	}
	target.WorkID = target.State.Contract.WorkItemID
	return target, nil
}

func compactProgressV2(progress ProgressV2) any {
	omitted := map[string]int{}
	clip := func(value, field string, size int) string {
		if len(value) <= size {
			return value
		}
		end := size
		for end > 0 && !utf8.ValidString(value[:end]) {
			end--
		}
		omitted[field+"_bytes"] += len(value) - end
		return value[:end]
	}
	completed := []string{}
	for i, value := range progress.Completed {
		if i >= 5 {
			omitted["completed"]++
			continue
		}
		completed = append(completed, clip(value, "completed", 512))
	}
	return map[string]any{"summary": clip(progress.Summary, "summary", 1024), "completed": completed, "next_action": clip(progress.NextAction, "next_action", 1024), "omitted": omitted}
}

func compactSignedStatusV2(state *a.SignedContractMetadata) any {
	if state == nil {
		return nil
	}
	return map[string]any{"contract_id": state.ID, "contract_kind": state.Kind, "status": state.Status, "effective_status": state.EffectiveStatus, "work_item_lifecycle": state.WorkItemLifecycle, "current": state.Current, "lease_valid": state.LeaseValid, "expires_at": state.ExpiresAt, "review_case_id": state.ReviewCaseID, "submission_id": state.SubmissionID}
}

func historicalSpecificationV2(ctx context.Context, profile ProfileV2, client *sdk.Client, target workTargetV2, kind string) (any, error) {
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: "specification", ContractKind: kind, ID: target.ID})
	if e != nil {
		return nil, e
	}
	if state.Envelope == nil {
		return nil, fmt.Errorf("issued historical specification unavailable")
	}
	document, e := signing.ToDocument(*state.Envelope)
	if e != nil {
		return nil, e
	}
	raw, e := verifyProfileDocumentV2(profile, document, signing.ContractSpec)
	if e != nil {
		return nil, e
	}
	var binding signing.SpecPayload[json.RawMessage]
	if e = signing.DecodeHistoricalPayload(raw, &binding, 128<<10); e != nil {
		return nil, e
	}
	if (binding.ProtocolVersion != 3 && binding.ProtocolVersion != 2) || binding.ServerID != profile.Binding.ServerID.String() || binding.TenantID != target.Scope.TenantID.String() || binding.OutcomeID != target.Scope.OutcomeID.String() || binding.ContractKind != kind || binding.ContractID != target.ID.String() || binding.WorkItemID != target.WorkID.String() {
		return nil, fmt.Errorf("historical specification target differs")
	}
	if target.State.Contract == nil || signing.Digest(raw) != target.State.Contract.SpecDigest {
		return nil, fmt.Errorf("historical specification digest differs from contract metadata")
	}
	if kind == "execution" {
		var spec signing.SpecPayload[d.SignedWorkSpec]
		if e = signing.DecodeHistoricalPayload(raw, &spec, 128<<10); e != nil {
			return nil, e
		}
		return spec.Spec, nil
	}
	var spec signing.SpecPayload[a.SignedReviewSpec]
	if e = signing.DecodeHistoricalPayload(raw, &spec, 128<<10); e != nil {
		return nil, e
	}
	return reviewAgentSpecificationV2(profile, ContractViewV2{ReviewSpecification: &spec})
}

func projectedDraftAttentionV2(file ContractFileV2, view ContractViewV2) []string {
	items := []string{}
	if file.Execution == nil {
		if strings.TrimSpace(file.Review.Material.Reason) == "" {
			items = append(items, "Set the review decision and reason.")
		}
		return items
	}
	m := file.Execution.Material
	if strings.TrimSpace(m.Result.Summary) == "" {
		items = append(items, "Record the result summary with work result set.")
	}
	for _, criterion := range view.WorkSpecification.Spec.Criteria {
		if !criterion.Required || criterion.Status != d.CriterionStatusActive {
			continue
		}
		linked := false
		for _, evidence := range m.Result.CriterionEvidence {
			linked = linked || evidence.CriterionID == criterion.ID && evidence.CriterionRevision == criterion.Revision && len(evidence.EvidenceIDs) > 0
		}
		for _, evidence := range m.CriterionLocalEvidence {
			linked = linked || evidence.CriterionID == criterion.ID && uint64(evidence.CriterionRevision) == uint64(criterion.Revision) && len(evidence.EvidenceLocalKeys) > 0
		}
		if !linked && criterion.VerificationMode != d.VerificationModeAttestation {
			items = append(items, "Attach evidence to criterion: "+criterion.Title)
		}
	}
	for _, finding := range view.WorkSpecification.Spec.CorrectionFindings {
		addressed := false
		for _, response := range m.CorrectionResponses {
			addressed = addressed || response.FindingID == finding.ID
		}
		if !addressed {
			items = append(items, "Address issued correction finding: "+finding.ID.String())
		}
	}
	return items
}

func queryWorkV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, identity a.SigningIdentityView, kind string, o options) (Output, error) {
	result := Output{Operation: o.args[0] + " " + o.args[1]}
	if e := requireActivityCapabilityV2(ctx, client, true); e != nil {
		return result, e
	}
	target, e := resolveWorkTargetV2(ctx, w, profile, client, kind, o, 2)
	if e != nil {
		return result, e
	}
	if target.File != nil && kind == "review" {
		target.View, e = expandReviewSpecificationV2(ctx, profile, client, target.View)
		if e != nil {
			return result, e
		}
	}
	result.ContractID, result.WorkItemID, result.OutcomeID = target.ID, target.WorkID, target.Scope.OutcomeID
	attention, e := client.GetWorkAttention(ctx, target.Scope, target.WorkID)
	if e != nil {
		return result, e
	}
	data := map[string]any{"profile": profile.Name, "status": compactSignedStatusV2(target.State.Contract), "attention": attention, "local_material_present": target.File != nil}
	result.Data = data
	if target.File != nil {
		file := *target.File
		digest, e := file.DraftDigest()
		if e != nil {
			return result, e
		}
		data["draft_digest"], data["draft_attention"] = digest, projectedDraftAttentionV2(file, target.View)
		data["pending_final_return"] = file.Local.Pending != nil
		if o.args[1] == "context" {
			data["draft"] = compactDraftV2(file)
		}
		if file.Execution != nil {
			data["progress"], data["completion_intent"] = compactProgressV2(file.Execution.Progress), file.Execution.CompletionIntent
			data["material"] = map[string]any{"summary": file.Execution.Material.Result.Summary, "evidence_count": len(file.Execution.Material.Result.EvidenceIDs), "artifact_count": len(file.Execution.Material.Result.Artifacts), "assessment_count": len(file.Execution.Material.Assessments), "correction_response_count": len(file.Execution.Material.CorrectionResponses)}
			if o.args[1] == "context" {
				delete(data["material"].(map[string]any), "summary")
			}
		} else {
			data["progress"], data["decision"] = compactProgressV2(file.Review.Progress), file.Review.Decision
		}
		if o.args[1] == "context" || o.args[1] == "show" {
			if kind == "execution" {
				data["specification"] = target.View.WorkSpecification.Spec
			} else {
				data["specification"], e = reviewAgentSpecificationV2(profile, target.View)
				if e != nil {
					return result, e
				}
			}
		}
	} else if o.args[1] == "context" || o.args[1] == "show" {
		if !target.ID.IsZero() {
			data["specification"], e = historicalSpecificationV2(ctx, profile, client, target, kind)
			if e != nil {
				return result, e
			}
		}
	}
	if o.args[1] == "context" || o.args[1] == "show" && target.File == nil {
		if e = populateWorkContextV2(ctx, client, target, o, data); e != nil {
			return result, e
		}
		if kind == "execution" && target.State.Contract != nil && target.State.Contract.SubmissionID != nil {
			data["accepted_result"], e = acceptedWorkResultV2(ctx, profile, client, target)
			if e != nil {
				return result, e
			}
		}
	}
	if o.args[1] != "check" {
		return result, nil
	}
	problems := []string{}
	if target.File == nil {
		problems = append(problems, "No local execution/review draft is available for signing.")
	} else {
		file, view := *target.File, target.View
		if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
			problems = append(problems, "A return is frozen or accepted; send/recover the original intention.")
		}
		for _, pending := range profile.Local.PendingOperations {
			if pending.ContractID != nil && *pending.ContractID == target.ID {
				problems = append(problems, "An unresolved operation must be recovered before signing.")
			}
		}
		if e = validateReturnDraftV2(profile, file, view); e != nil {
			problems = append(problems, e.Error())
		}
		grant, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: "authority", ID: target.ID, ContractKind: kind})
		if e != nil {
			return result, e
		}
		var review a.SignedStateResult
		if kind == "review" {
			review, e = client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: "case", ID: d.ID(view.Authority.ReviewCaseID)})
			if e != nil {
				return result, e
			}
		}
		if e = liveReturnPreflightV2(profile, file, view, identity, target.State, grant, review); e != nil {
			problems = append(problems, e.Error())
		}
	}
	data["technical_ready"], data["checks"], data["acceptance_authority"] = len(problems) == 0, problems, "The server validates delivery; evidence does not approve criteria and required review remains independent."
	if len(problems) > 0 {
		return result, &exitError{7, fmt.Errorf("delivery needs attention; inspect checks and recover frozen intentions before changing material")}
	}
	return result, nil
}

func historyWorkV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, kind string, o options) (Output, error) {
	position := 2
	section := "history"
	if o.args[1] == "evidence" || o.args[1] == "artifact" {
		position, section = 3, o.args[1]
		if len(o.args) < 3 || o.args[2] != "list" {
			return Output{}, usage("evidence/artifact list [reference]")
		}
	}
	result := Output{Operation: o.args[0] + " " + section}
	if e := requireActivityCapabilityV2(ctx, client, false); e != nil {
		return result, e
	}
	target, e := resolveWorkTargetV2(ctx, w, profile, client, kind, o, position)
	if e != nil {
		return result, e
	}
	limit, e := operationLimitV2(o)
	if e != nil {
		return result, e
	}
	order := o.values["order"]
	if order != "" && order != "asc" && order != "desc" {
		return result, usage("--order must be asc or desc")
	}
	contract := target.ID
	if kind == "review" {
		contract = ""
	}
	page, e := client.ListWorkActivity(ctx, target.Scope, ports.WorkActivityFilter{WorkItemID: target.WorkID, ContractID: contract, Descending: order == "desc"}, limit, o.values["cursor"])
	result.OutcomeID, result.WorkItemID, result.ContractID = target.Scope.OutcomeID, target.WorkID, target.ID
	if e != nil {
		return result, e
	}
	if section == "history" {
		result.Data = page
		return result, nil
	}
	ids := []d.ID{}
	for _, activity := range page.Items {
		refs := activity.EvidenceRefs
		if section == "artifact" {
			refs = activity.ArtifactRefs
		}
		for _, id := range refs {
			ids = appendUniqueIDV2(ids, id)
		}
	}
	omitted := max(0, len(ids)-100)
	if len(ids) > 100 {
		ids = ids[:100]
	}
	items := []any{}
	for _, id := range ids {
		collection := "evidence"
		if section == "artifact" {
			collection = "artifacts"
		}
		raw, e := client.QueryJSON(ctx, outcomePath(target.Scope)+"/"+collection+"/"+id.String())
		if e != nil {
			return result, e
		}
		if section == "artifact" {
			var value a.ReadResult[d.Artifact]
			if e = json.Unmarshal(raw, &value); e != nil {
				return result, e
			}
			artifact := value.Value
			items = append(items, map[string]any{"id": artifact.ID, "name": artifact.Name, "uri": artifact.URI, "source_version": artifact.SourceVersion, "checksum": artifact.Checksum, "lifecycle": artifact.Lifecycle})
		} else {
			var value a.ReadResult[d.Evidence]
			if e = json.Unmarshal(raw, &value); e != nil {
				return result, e
			}
			evidence := value.Value
			items = append(items, map[string]any{"id": evidence.ID, "type": evidence.EvidenceType, "description": evidence.Description, "source": evidence.SourceRef, "lifecycle": evidence.Lifecycle})
		}
	}
	result.Data = map[string]any{"items": items, "next_cursor": page.NextCursor, "outcome_revision": page.OutcomeRevision, "history_through_revision": page.HistoryThroughRevision, "activity_page_limit": limit, "records_omitted": omitted, "pagination": "The cursor traverses activity pages; reduce --limit when records_omitted is nonzero."}
	return result, nil
}

func compactFocalContextV2(context a.WorkContext) map[string]any {
	omitted := map[string]int{}
	for key, count := range context.Omitted {
		omitted[key] = count
	}
	clip := func(text, field string, maximum int) string {
		if len(text) <= maximum {
			return text
		}
		end := maximum
		for end > 0 && !utf8.ValidString(text[:end]) {
			end--
		}
		omitted[field+"_bytes"] += len(text) - end
		return text[:end]
	}
	current := map[string]any{"work_item_id": context.Work.ID, "title": clip(context.Work.Title, "work.title", 512), "description": clip(context.Work.Description, "work.description", 2048), "lifecycle": context.Work.Lifecycle, "version": strconv.FormatUint(uint64(context.Work.Version), 10)}
	outcome := map[string]any{"id": context.Outcome.Ref.ID, "title": context.Outcome.Title, "lifecycle": context.Outcome.Lifecycle, "detail_omitted": context.Outcome.DetailOmitted}
	var objective any
	if context.Objective != nil {
		value := context.Objective
		objective = map[string]any{"id": value.ID, "title": clip(value.Title, "objective.title", 512), "description": clip(value.Description, "objective.description", 2048), "lifecycle": value.Lifecycle, "required_for_outcome": value.RequiredForOutcome}
	}
	dependencies := context.Dependencies
	if len(dependencies) > 10 {
		omitted["dependencies"] += len(dependencies) - 10
		dependencies = dependencies[:10]
	}
	blockers := []any{}
	isBlocked := false
	if context.Blocking != nil {
		isBlocked = context.Blocking.IsBlocked
		for i, item := range context.Blocking.ActiveBlockers {
			if i >= 10 {
				omitted["active_blockers"]++
				continue
			}
			value := item.Blocker
			blockers = append(blockers, map[string]any{"id": value.ID, "description": clip(value.Description, "blocker.description", 1024), "blocked_ref": value.BlockedRef, "cause_ref": value.CauseRef, "inherited": item.Inherited, "inherited_from": item.InheritedFrom})
		}
	}
	decisions := []any{}
	for i, decision := range context.Decisions {
		if i >= 5 {
			omitted["current_decisions"]++
			continue
		}
		decisions = append(decisions, map[string]any{"id": decision.ID, "title": clip(decision.Title, "decision.title", 512), "choice": clip(decision.ChosenAlternative, "decision.choice", 1024), "rationale": clip(decision.Rationale, "decision.rationale", 1024)})
	}
	truncated := context.Truncated
	for _, count := range omitted {
		truncated = truncated || count > 0
	}
	return map[string]any{"work": current, "outcome": outcome, "objective": objective, "dependencies": dependencies, "is_blocked": isBlocked, "active_blockers": blockers, "current_decisions": decisions, "outcome_revision": strconv.FormatUint(uint64(context.OutcomeRevision), 10), "evaluated_at": context.EvaluatedAt, "truncated": truncated, "omitted": omitted, "authority": "current coordination context; immutable instructions remain in specification"}
}

func populateWorkContextV2(ctx context.Context, client *sdk.Client, target workTargetV2, o options, data map[string]any) error {
	current, e := client.GetWorkContext(ctx, target.Scope, target.WorkID)
	if e != nil {
		return e
	}
	if current.Work.ID != target.WorkID || current.Work.Scope != target.Scope {
		return fmt.Errorf("focal context target differs")
	}
	data["context"] = compactFocalContextV2(current)
	if e = requireActivityCapabilityV2(ctx, client, false); e != nil {
		return e
	}
	limit := 5
	if o.values["limit"] != "" {
		limit, e = operationLimitV2(o)
		if e != nil {
			return e
		}
		limit = min(limit, 25)
	}
	page, e := client.ListWorkActivity(ctx, target.Scope, ports.WorkActivityFilter{WorkItemID: target.WorkID, Descending: true}, limit, "")
	if e != nil {
		return e
	}
	data["recent_activity"], data["activity_limit"] = compactRecentActivityV2(page), limit
	if page.NextCursor != "" {
		data["history_follow_up"] = map[string]any{"command": "work history", "work": target.WorkID, "outcome": target.Scope.OutcomeID, "order": "desc", "cursor": page.NextCursor, "limit": limit}
	}
	data["snapshot_consistency"] = "Each server resource has its own authoritative revision/evaluation time; focal context, attention and activity are read separately."
	return nil
}

func acceptedWorkResultV2(ctx context.Context, profile ProfileV2, client *sdk.Client, target workTargetV2) (any, error) {
	submissionID := *target.State.Contract.SubmissionID
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: "submission", ID: submissionID})
	if e != nil {
		return nil, e
	}
	if state.SubmissionID == nil || *state.SubmissionID != submissionID || state.AcceptanceFactID == nil || state.Scope == nil || *state.Scope != target.Scope {
		return nil, fmt.Errorf("accepted result submission target differs")
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(state.MaterialPayload)
	if e != nil || len(raw) > signing.MaxPayloadBytes || signing.Digest(raw) != state.SubmissionDigest {
		return nil, fmt.Errorf("accepted result payload/digest differs")
	}
	var material d.SignedResultMaterial
	if e = signing.DecodeStrict(raw, &material, signing.MaxPayloadBytes); e != nil {
		return nil, e
	}
	if material.ContractID != target.ID || material.WorkItemID != target.WorkID || material.SpecDigest != target.State.Contract.SpecDigest {
		return nil, fmt.Errorf("accepted result contract/specification target differs")
	}
	fact, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: target.Scope, Resource: "fact", ID: *state.AcceptanceFactID})
	if e != nil {
		return nil, e
	}
	if fact.Envelope == nil {
		return nil, fmt.Errorf("accepted result issuer receipt unavailable")
	}
	document, e := signing.ToDocument(*fact.Envelope)
	if e != nil {
		return nil, e
	}
	payload, e := verifyProfileDocumentV2(profile, document, signing.AcceptanceReceipt)
	if e != nil {
		return nil, e
	}
	var receipt signing.ReceiptPayload
	if e = signing.DecodeHistoricalPayload(payload, &receipt, signing.MaxPayloadBytes); e != nil {
		return nil, e
	}
	if !receipt.Accepted || !receipt.LocalObligationClosed || (receipt.ProtocolVersion != 3 && receipt.ProtocolVersion != 2) || receipt.ServerID != profile.Binding.ServerID.String() || receipt.TenantID != target.Scope.TenantID.String() || receipt.OutcomeID != target.Scope.OutcomeID.String() || receipt.ContractID != target.ID.String() || receipt.WorkItemID != target.WorkID.String() || receipt.PrincipalID != target.State.Contract.HolderPrincipalID || receipt.SubmissionID != submissionID.String() || receipt.SubmissionDigest != state.SubmissionDigest || receipt.SignerKeyID != document.Proof.KeyID {
		return nil, fmt.Errorf("accepted result issuer receipt target/digest differs")
	}
	view := compactDraftV2(ContractFileV2{Execution: &ExecutionDraftV2{Material: a.SignedReturnMaterial{Result: material}}, Local: ContractLocalV2{AcceptanceReceipt: &document}})
	delete(view, "expand")
	view["submission_id"], view["submission_digest"], view["issuer_verified"], view["authority"] = submissionID, state.SubmissionDigest, true, "accepted execution material; this read does not renew execution authority or approve parent Outcomes"
	return view, nil
}

func listWorkV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, o options) (Output, error) {
	result := Output{Operation: "work list"}
	if len(o.args) != 2 {
		return result, usage("work list [--outcome reference] [--status persisted-status] [--holder Principal]")
	}
	if e := requireActivityCapabilityV2(ctx, client, false); e != nil {
		return result, e
	}
	scope, e := resolveOutcomeV2(ctx, w, profile, client, o)
	if e != nil {
		return result, e
	}
	limit, e := operationLimitV2(o)
	if e != nil {
		return result, e
	}
	holder := o.values["holder"]
	if holder == "" {
		holder = profile.Binding.PrincipalID
	}
	status := d.ContractStatus(o.values["status"])
	if status != "" && status != d.ContractActive && status != d.ContractDelivered && status != d.ContractCompleted && status != d.ContractExpired && status != d.ContractRevoked {
		return result, usage("--status must be active, delivered, completed, expired or revoked (persisted status)")
	}
	var work d.ID
	if value := o.values["work"]; value != "" {
		work, e = d.ParseID(value)
		if e != nil {
			return result, e
		}
	}
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "executions", HolderPrincipalID: holder, WorkItemID: work, Status: status, Limit: limit, Cursor: o.values["cursor"]})
	if e != nil {
		return result, e
	}
	items := []a.SignedExecutionHeader{}
	filtered := 0
	for _, header := range state.Executions {
		if holder == profile.Binding.PrincipalID && header.CredentialID != profile.Binding.CredentialID {
			filtered++
			continue
		}
		items = append(items, header)
	}
	ids, e := contractIDsV2(w, profile.Name)
	if e != nil {
		return result, e
	}
	local := []any{}
	for _, id := range ids {
		legacy, isLegacy, e := readLegacyUnsignedV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if isLegacy {
			local = append(local, legacyUnsignedSummaryV2(legacy))
			continue
		}
		file, view, _, _, e := loadContractV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if view.Authority.ContractKind != "execution" || view.Authority.OutcomeID != scope.OutcomeID.String() {
			continue
		}
		local = append(local, map[string]any{"contract_id": id, "work_item_id": view.Authority.WorkItemID, "title": view.WorkSpecification.Spec.Title, "pending_final_return": file.Local.Pending != nil})
	}
	result.OutcomeID = scope.OutcomeID
	result.Data = map[string]any{"profile": profile.Name, "items": items, "local_contracts": local, "next_cursor": state.NextCursor, "search_complete": state.SearchComplete, "outcome_revision": state.OutcomeRevision, "filtered_other_credentials": filtered, "status_filter": "persisted status; effective_status also reflects lease expiry", "authority": "read-only server metadata; local absence after accepted cleanup does not erase history"}
	return result, nil
}
