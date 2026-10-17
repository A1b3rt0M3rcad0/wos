package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// CLI edits hold stable profile/contract locks and compare the document bytes
// before publication. Neither issuer documents nor frozen intentions are editable.
func mutateMaterialV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, id d.ID, kind, expected string, fn func(*ContractFileV2, ContractViewV2) error) (string, error) {
	unlock, e := w.LockV2(ctx, profile.Name, "")
	if e != nil {
		return "", e
	}
	defer unlock()
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		return "", e
	}
	if e = current.VerifyBinding(token); e != nil {
		return "", e
	}
	if current.Local.BindingMAC != profile.Local.BindingMAC {
		return "", fmt.Errorf("profile binding changed; material preserved")
	}
	for _, intent := range current.Local.PendingOperations {
		if intent.ContractID != nil && *intent.ContractID == id {
			return "", fmt.Errorf("contract has a frozen unresolved intention; recover it before editing")
		}
	}
	release, e := w.LockV2(ctx, profile.Name, id.String())
	if e != nil {
		return "", e
	}
	defer release()
	file, view, path, raw, e := loadContractV2(w, current, id)
	if e != nil {
		return "", e
	}
	if view.Authority.ProtocolVersion != signing.ProtocolVersion {
		return "", fmt.Errorf("historical protocol-2 contract is read-only; acquire a new protocol-3 contract")
	}
	if view.Authority.ContractKind != kind {
		return "", usage("contract kind differs from command")
	}
	if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
		return "", fmt.Errorf("final return is frozen or accepted; material preserved")
	}
	digest, e := file.DraftDigest()
	if e != nil {
		return "", e
	}
	if expected != "" && digest != expected {
		return "", fmt.Errorf("local_version_conflict: material changed; inspect it before retrying")
	}
	issued, e := json.Marshal(file.Issued)
	if e != nil {
		return "", e
	}
	local, e := json.Marshal(file.Local)
	if e != nil {
		return "", e
	}
	if e = fn(&file, view); e != nil {
		return "", e
	}
	afterIssued, _ := json.Marshal(file.Issued)
	afterLocal, _ := json.Marshal(file.Local)
	if !bytes.Equal(issued, afterIssued) || !bytes.Equal(local, afterLocal) {
		return "", fmt.Errorf("issued documents are immutable")
	}
	if _, e = file.VerifyIssued(current); e != nil {
		return "", e
	}
	if file.Execution != nil {
		result := file.Execution.Material.Result
		if result.ContractID.String() != view.Authority.ContractID || result.WorkItemID.String() != view.Authority.WorkItemID || result.SpecDigest != view.Authority.SpecDigest {
			return "", fmt.Errorf("result material binding differs from issued contract")
		}
	}
	encoded, e := EncodeV2Document(file)
	if e != nil {
		return "", e
	}
	var checked ContractFileV2
	if e = DecodeV2Document(encoded, &checked); e != nil {
		return "", e
	}
	if e = w.WriteV2(path, file, signing.Digest(raw)); e != nil {
		return "", e
	}
	return file.DraftDigest()
}

func draftCriteriaV2(profile ProfileV2, view ContractViewV2) ([]d.SignedCriterion, error) {
	if view.WorkSpecification != nil {
		return view.WorkSpecification.Spec.Criteria, nil
	}
	if view.ReviewSpecification == nil || view.ReviewSpecification.Spec.IssuedWorkSpecification == nil {
		return nil, fmt.Errorf("issued work criteria are referenced remotely; expand the original specification before assessing")
	}
	raw, e := verifyProfileDocumentV2(profile, *view.ReviewSpecification.Spec.IssuedWorkSpecification, signing.ContractSpec)
	if e != nil {
		return nil, e
	}
	if signing.Digest(raw) != view.ReviewSpecification.Spec.IssuedWorkSpecDigest {
		return nil, fmt.Errorf("reviewed specification digest differs")
	}
	var spec signing.SpecPayload[d.SignedWorkSpec]
	if e = signing.DecodeHistoricalPayload(raw, &spec, 128<<10); e != nil {
		return nil, e
	}
	if spec.ContractKind != "execution" || spec.ServerID != view.Authority.ServerID || spec.TenantID != view.Authority.TenantID || spec.OutcomeID != view.Authority.OutcomeID || spec.WorkItemID != view.Authority.WorkItemID {
		return nil, fmt.Errorf("reviewed specification target differs")
	}
	return spec.Spec.Criteria, nil
}

func criterionReferenceV2(profile ProfileV2, view ContractViewV2, ref string) (d.SignedCriterion, error) {
	criteria, e := draftCriteriaV2(profile, view)
	if e != nil {
		return d.SignedCriterion{}, e
	}
	matches := []d.SignedCriterion{}
	for _, criterion := range criteria {
		if criterion.Status == d.CriterionStatusActive && matchesReferenceV2(ref, criterion.ID.String(), criterion.Title) {
			matches = append(matches, criterion)
		}
	}
	if len(matches) != 1 {
		return d.SignedCriterion{}, usage(fmt.Sprintf("criterion reference matched %d active criteria; choose a unique title or ID", len(matches)))
	}
	return matches[0], nil
}

func evidenceIDsOptionV2(value string) ([]d.ID, error) {
	ids := []d.ID{}
	if value == "" {
		return ids, nil
	}
	for _, part := range strings.Split(value, ",") {
		id, e := d.ParseID(strings.TrimSpace(part))
		if e != nil {
			return nil, usage("--evidence requires comma-separated Evidence IDs")
		}
		for _, existing := range ids {
			if existing == id {
				return nil, usage("duplicate Evidence ID")
			}
		}
		ids = append(ids, id)
	}
	if len(ids) > 100 {
		return nil, usage("at most 100 Evidence IDs")
	}
	return ids, nil
}

func assessmentEvidenceV2(value string, evidence []a.SyncEvidenceInput) ([]d.ID, []string, error) {
	remote := []string{}
	local := []string{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if key, ok := strings.CutPrefix(part, "local:"); ok {
			found := false
			for _, item := range evidence {
				found = found || item.LocalKey == key
			}
			if !found {
				return nil, nil, usage("unknown local Evidence key; use the key returned by evidence add")
			}
			for _, existing := range local {
				if existing == key {
					return nil, nil, usage("duplicate local Evidence key")
				}
			}
			local = append(local, key)
		} else {
			remote = append(remote, part)
		}
	}
	ids, e := evidenceIDsOptionV2(strings.Join(remote, ","))
	return ids, local, e
}

func materialCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, kind string, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args[:min(len(o.args), 3)], " ")}
	if len(o.args) < 3 {
		return result, usage("result/progress/decision set or assessment/finding/correction add")
	}
	sub, verb := o.args[1], o.args[2]
	if sub == "finding" && o.values["criterion"] != "" && o.values["requirement"] != "" {
		return result, usage("a finding must select either --criterion or --requirement")
	}
	if (sub == "result" || sub == "progress" || sub == "decision") && verb != "set" || (sub == "assessment" || sub == "finding" || sub == "correction") && verb != "add" && verb != "set" && verb != "remove" && verb != "list" {
		return result, usage("use set for result/progress/decision; add|set|remove|list for assessment/finding/correction")
	}
	id, e := contractOptionV2(w, profile, kind, o, 3)
	if e != nil {
		return result, e
	}
	result.ContractID = id
	if verb == "list" {
		file, view, _, _, e := loadContractV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if view.Authority.ContractKind != kind {
			return result, usage("contract kind differs")
		}
		items := []any{}
		switch sub {
		case "assessment":
			values := file.draftAssessmentsV2()
			for _, item := range values {
				items = append(items, item)
			}
		case "finding":
			if file.Review == nil {
				return result, usage("finding list requires a review contract")
			}
			for _, item := range file.Review.Material.Findings {
				items = append(items, item)
			}
		case "correction":
			if file.Execution == nil {
				return result, usage("correction list requires an execution contract")
			}
			for _, item := range file.Execution.Material.CorrectionResponses {
				items = append(items, item)
			}
		}
		result.Data, e = materialPageV2(file, sub, items, o)
		return result, e
	}
	expanded, e := materialReviewViewV2(ctx, w, profile, client, id, kind)
	if e != nil {
		return result, e
	}
	digest, e := mutateMaterialV2(ctx, w, profile, token, id, kind, o.values["expected-draft"], func(file *ContractFileV2, view ContractViewV2) error {
		if expanded != nil {
			if view.Authority.SpecDigest != expanded.Authority.SpecDigest {
				return fmt.Errorf("review specification changed during material preflight")
			}
			view = *expanded
		}
		text := func(value, label string) (string, error) {
			if strings.TrimSpace(value) == "" || len(value) > 16384 {
				return "", usage(label + " must be nonempty and at most 16384 bytes")
			}
			return value, nil
		}
		switch sub {
		case "result":
			if file.Execution == nil {
				return usage("result set requires an execution contract")
			}
			summary, e := text(o.values["summary"], "--summary")
			if e != nil {
				return e
			}
			file.Execution.Material.Result.Summary = summary
			if mode := o.values["completion"]; mode != "" {
				if mode != "auto" && mode != "direct" && mode != "review" {
					return usage("--completion must be auto, direct or review")
				}
				file.Execution.CompletionIntent = mode
			}
		case "progress":
			summary, e := text(o.values["summary"], "--summary")
			if e != nil {
				return e
			}
			if len(o.values["next-action"]) > 16384 {
				return usage("--next-action exceeds 16384 bytes")
			}
			var progress *ProgressV2
			if file.Execution != nil {
				progress = &file.Execution.Progress
			} else {
				progress = &file.Review.Progress
			}
			progress.Summary, progress.NextAction = summary, o.values["next-action"]
		case "decision":
			if file.Review == nil {
				return usage("decision set requires a review contract")
			}
			decision := o.values["decision"]
			if decision != "approved" && decision != "changes_requested" && decision != "inconclusive" {
				return usage("--decision must be approved, changes_requested or inconclusive")
			}
			reason, e := text(o.values["reason"], "--reason")
			if e != nil {
				return e
			}
			if decision != "approved" && len(file.Review.Material.Assessments) > 0 {
				return usage("existing assessments require an approved decision; do not discard assessments implicitly")
			}
			file.Review.Decision, file.Review.Material.Reason = decision, reason
		case "assessment":
			criterion, e := criterionReferenceV2(profile, view, o.values["criterion"])
			if e != nil {
				return e
			}
			var assessments *[]a.SignedAssessmentInput
			if file.Execution != nil {
				assessments = &file.Execution.Material.Assessments
			} else {
				assessments = &file.Review.Material.Assessments
			}
			index := -1
			for i, existing := range *assessments {
				if existing.CriterionID == criterion.ID {
					index = i
				}
			}
			if verb == "remove" {
				if index < 0 {
					return usage("criterion has no draft assessment")
				}
				*assessments = append((*assessments)[:index], (*assessments)[index+1:]...)
				return nil
			}
			value := d.AssessmentResult(o.values["result"])
			if !value.Valid() {
				return usage("--result must be a valid criterion assessment result")
			}
			rationale, e := text(o.values["rationale"], "--rationale")
			if e != nil {
				return e
			}
			evidence := file.draftEvidenceV2()
			ids, local, e := assessmentEvidenceV2(o.values["evidence"], evidence)
			if e != nil {
				return e
			}
			input := a.SignedAssessmentInput{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), Result: value, Rationale: rationale, EvidenceIDs: ids, EvidenceLocalKeys: local}
			if file.Review != nil {
				if file.Review.Decision != "approved" {
					return usage("set an approved review decision before adding assessments")
				}
			}
			if index >= 0 {
				if verb == "add" {
					return usage("criterion already assessed; use assessment set or remove")
				}
				(*assessments)[index] = input
				return nil
			}
			if len(*assessments) >= 100 {
				return usage("assessment limit is 100")
			}
			*assessments = append(*assessments, input)
		case "finding":
			if file.Review == nil {
				return usage("finding add requires a review contract")
			}
			index := -1
			if verb != "add" {
				for i, finding := range file.Review.Material.Findings {
					if matchesReferenceV2(o.values["finding"], finding.ID.String(), finding.Description) {
						if index >= 0 {
							return usage("ambiguous --finding; select an exact draft finding ID")
						}
						index = i
					}
				}
				if index < 0 {
					return usage("draft finding was not found")
				}
				if verb == "remove" {
					file.Review.Material.Findings = append(file.Review.Material.Findings[:index], file.Review.Material.Findings[index+1:]...)
					return nil
				}
			}
			description, e := text(o.values["description"], "--description")
			if e != nil {
				return e
			}
			id, e := newLocalIDV2()
			if e != nil {
				return e
			}
			finding := d.SignedFinding{ID: id, RequirementRef: o.values["requirement"], Description: description}
			if index >= 0 {
				finding = file.Review.Material.Findings[index]
				finding.Description = description
				if o.values["requirement"] != "" {
					finding.RequirementRef, finding.CriterionID = o.values["requirement"], nil
				}
			}
			if o.values["criterion"] != "" {
				criterion, e := criterionReferenceV2(profile, view, o.values["criterion"])
				if e != nil {
					return e
				}
				finding.CriterionID = &criterion.ID
				finding.RequirementRef = ""
			}
			if e = finding.Validate(); e != nil {
				return e
			}
			if len(file.Review.Material.Findings) >= 100 && index < 0 {
				return usage("finding limit is 100")
			}
			if index >= 0 {
				file.Review.Material.Findings[index] = finding
			} else {
				file.Review.Material.Findings = append(file.Review.Material.Findings, finding)
			}
			if e = validateReturnDraftV2(profile, *file, view); e != nil {
				return e
			}
		case "correction":
			if file.Execution == nil {
				return usage("correction add requires an execution contract")
			}
			matches := []d.SignedFinding{}
			for _, finding := range view.WorkSpecification.Spec.CorrectionFindings {
				if matchesReferenceV2(o.values["finding"], finding.ID.String(), "") {
					matches = append(matches, finding)
				}
			}
			if len(matches) != 1 {
				return usage("--finding must select exactly one issued correction finding")
			}
			index := -1
			for i, existing := range file.Execution.Material.CorrectionResponses {
				if existing.FindingID == matches[0].ID {
					index = i
				}
			}
			if verb == "remove" {
				if index < 0 {
					return usage("finding has no draft response")
				}
				file.Execution.Material.CorrectionResponses = append(file.Execution.Material.CorrectionResponses[:index], file.Execution.Material.CorrectionResponses[index+1:]...)
				return nil
			}
			summary, e := text(o.values["summary"], "--summary")
			if e != nil {
				return e
			}
			ids, e := evidenceIDsOptionV2(o.values["evidence"])
			if e != nil {
				return e
			}
			response := a.SignedFindingResponse{FindingID: matches[0].ID, Summary: summary, EvidenceIDs: ids}
			if index >= 0 {
				if verb == "add" {
					return usage("finding already addressed; use correction set or remove")
				}
				file.Execution.Material.CorrectionResponses[index] = response
			} else {
				file.Execution.Material.CorrectionResponses = append(file.Execution.Material.CorrectionResponses, response)
			}
		default:
			return usage("unknown editable material command")
		}
		return nil
	})
	if e == nil {
		result.Data = map[string]any{"contract_id": id, "draft_digest": digest, "local_saved": true, "remote_committed": false}
	}
	return result, e
}

func (file ContractFileV2) draftEvidenceV2() []a.SyncEvidenceInput {
	if file.Review != nil {
		return file.Review.Material.Evidence
	}
	if file.Execution != nil {
		return file.Execution.Material.Evidence
	}
	return nil
}

func (file ContractFileV2) draftAssessmentsV2() []a.SignedAssessmentInput {
	if file.Review != nil {
		return file.Review.Material.Assessments
	}
	if file.Execution != nil {
		return file.Execution.Material.Assessments
	}
	return nil
}

func materialPageV2(file ContractFileV2, section string, items []any, o options) (any, error) {
	limit, e := operationLimitV2(o)
	if e != nil {
		return nil, e
	}
	digest, e := file.DraftDigest()
	if e != nil {
		return nil, e
	}
	identity, e := signing.Canonical([]string{file.Local.Profile, signing.Digest(file.Issued.Specification.Payload), digest, section})
	if e != nil {
		return nil, e
	}
	binding := signing.Digest(identity)
	offset := 0
	if cursor := o.values["cursor"]; cursor != "" {
		index := strings.LastIndex(cursor, ":")
		if index < 0 || cursor[:index] != binding {
			return nil, usage("local material cursor is stale or belongs to another section; start a fresh list")
		}
		offset, e = strconv.Atoi(cursor[index+1:])
		if e != nil || offset < 0 || offset > len(items) {
			return nil, usage("invalid local material cursor")
		}
	}
	end := min(offset+limit, len(items))
	for {
		next := ""
		if end < len(items) {
			next = fmt.Sprintf("%s:%d", binding, end)
		}
		page := map[string]any{"items": items[offset:end], "next_cursor": next, "draft_digest": digest, "local_staged": true, "remote_committed": false, "total": len(items)}
		raw, e := json.Marshal(page)
		if e != nil {
			return nil, e
		}
		if len(raw) <= 64<<10 {
			return page, nil
		}
		if end <= offset+1 {
			return nil, fmt.Errorf("one material record exceeds compact output bound")
		}
		end--
	}
}

// Review observations are part of the reviewer's signed return, rather than an
// executor's activity stream. The CLI stages them; only finish registers them.
func reviewDocumentCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args[:min(len(o.args), 3)], " ")}
	if len(o.args) < 3 || o.args[1] != "evidence" && o.args[1] != "artifact" || o.args[2] != "add" && o.args[2] != "list" && o.args[2] != "remove" {
		return result, usage("review evidence/artifact add|list|remove [reference]")
	}
	id, e := contractOptionV2(w, profile, "review", o, 3)
	if e != nil {
		return result, e
	}
	result.ContractID = id
	if o.args[2] == "list" {
		file, view, _, _, e := loadContractV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if view.Authority.ContractKind != "review" {
			return result, usage("review contract required")
		}
		items := []any{}
		if o.args[1] == "artifact" {
			for _, item := range file.Review.Material.Artifacts {
				items = append(items, item)
			}
		} else {
			for _, item := range file.Review.Material.Evidence {
				items = append(items, item)
			}
		}
		result.Data, e = materialPageV2(file, o.args[1], items, o)
		return result, e
	}
	key, e := newLocalIDV2()
	if e != nil {
		return result, e
	}
	localKey := "review_" + strings.ReplaceAll(key.String(), "-", "")
	expanded, e := materialReviewViewV2(ctx, w, profile, client, id, "review")
	if e != nil {
		return result, e
	}
	digest, e := mutateMaterialV2(ctx, w, profile, token, id, "review", o.values["expected-draft"], func(file *ContractFileV2, view ContractViewV2) error {
		if expanded != nil {
			if view.Authority.SpecDigest != expanded.Authority.SpecDigest {
				return fmt.Errorf("review specification changed during material preflight")
			}
			view = *expanded
		}
		material := &file.Review.Material
		if o.args[2] == "remove" {
			return removeReviewDocumentV2(material, o.args[1], o)
		}
		additional := 1
		if o.args[1] == "evidence" {
			additional = 2
		}
		if len(material.Evidence)+len(material.Artifacts)+len(material.EvidenceLinks)+additional > 100 {
			return usage("review documentary material limit is 100")
		}
		if strings.TrimSpace(o.values["uri"]) == "" {
			return usage("--uri is required")
		}
		scope := d.Scope{TenantID: profile.Binding.TenantID, OutcomeID: d.ID(view.Authority.OutcomeID)}
		if o.args[1] == "artifact" {
			if strings.TrimSpace(o.values["name"]) == "" || strings.TrimSpace(o.values["artifact-type"]) == "" {
				return usage("artifact add requires --name and --artifact-type")
			}
			material.Artifacts = append(material.Artifacts, a.SyncArtifactInput{LocalKey: localKey, Artifact: a.RegisterArtifactCommand{Scope: scope, Name: o.values["name"], ArtifactType: o.values["artifact-type"], URI: o.values["uri"], SourceVersion: o.values["source-version"], Checksum: o.values["content-hash"]}})
			return nil
		}
		if strings.TrimSpace(o.values["description"]) == "" || len(o.values["description"]) > 16384 {
			return usage("evidence --description must be nonempty and at most 16384 bytes")
		}
		kind := d.EvidenceType(o.values["type"])
		if kind == "" {
			kind = d.EvidenceTypeInspection
		}
		if !kind.Valid() || kind == d.EvidenceTypeMeasurement {
			return usage("unsupported evidence type; typed measurements require structured material")
		}
		material.Evidence = append(material.Evidence, a.SyncEvidenceInput{LocalKey: localKey, Evidence: a.RegisterEvidenceCommand{Scope: scope, Description: o.values["description"], EvidenceType: kind, SourceRef: d.SourceReference{Provider: "wos-cli", URI: o.values["uri"]}, CapturedAt: time.Now().UTC(), SourceVersion: o.values["source-version"], Checksum: o.values["content-hash"]}})
		link := a.SyncEvidenceLinkRecord{Scope: scope, TargetRef: d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: d.ID(view.Authority.WorkItemID)}, Stance: d.EvidenceStanceSupports, Rationale: o.values["description"]}
		if ref := o.values["criterion"]; ref != "" {
			criterion, e := criterionReferenceV2(profile, view, ref)
			if e != nil {
				return e
			}
			link.CriterionID = &criterion.ID
			material.CriterionLocalEvidence = append(material.CriterionLocalEvidence, a.SignedLocalCriterionEvidence{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), EvidenceLocalKeys: []string{localKey}})
		}
		material.EvidenceLinks = append(material.EvidenceLinks, a.SyncEvidenceLinkInput{LocalKey: localKey + "_link", EvidenceLocalKey: localKey, Link: link})
		return nil
	})
	if e == nil {
		result.Data = map[string]any{"local_key": localKey, "draft_digest": digest, "local_saved": true, "remote_committed": false, "requires_action": "review finish submits this material under the original signed review authority"}
		if o.args[2] == "remove" {
			result.Data = map[string]any{"removed": true, "draft_digest": digest, "local_saved": true, "remote_committed": false}
		}
	}
	return result, e
}

func removeReviewDocumentV2(material *a.SignedReviewMaterial, section string, o options) error {
	if section == "artifact" {
		ref := o.values["name"]
		index := -1
		for i, item := range material.Artifacts {
			if ref == item.LocalKey || strings.EqualFold(ref, item.Artifact.Name) {
				if index >= 0 {
					return usage("ambiguous local artifact; use its exact local key with --name")
				}
				index = i
			}
		}
		if index < 0 {
			return usage("local staged artifact not found; select --name LOCAL_KEY")
		}
		key := material.Artifacts[index].LocalKey
		for _, evidence := range material.Evidence {
			if evidence.ArtifactLocalKey == key {
				return usage("artifact is referenced by staged evidence; remove that evidence explicitly first")
			}
		}
		material.Artifacts = append(material.Artifacts[:index], material.Artifacts[index+1:]...)
		return nil
	}
	key := strings.TrimPrefix(o.values["evidence"], "local:")
	index := -1
	for i, item := range material.Evidence {
		if item.LocalKey == key {
			index = i
		}
	}
	if index < 0 {
		return usage("local staged evidence not found; select --evidence local:LOCAL_KEY")
	}
	for _, assessment := range material.Assessments {
		for _, ref := range assessment.EvidenceLocalKeys {
			if ref == key {
				return usage("evidence is referenced by a draft assessment; set or remove that assessment explicitly first")
			}
		}
	}
	material.Evidence = append(material.Evidence[:index], material.Evidence[index+1:]...)
	links := material.EvidenceLinks[:0]
	for _, link := range material.EvidenceLinks {
		if link.EvidenceLocalKey != key {
			links = append(links, link)
		}
	}
	material.EvidenceLinks = links
	criteria := material.CriterionLocalEvidence[:0]
	for _, criterion := range material.CriterionLocalEvidence {
		keys := []string{}
		for _, ref := range criterion.EvidenceLocalKeys {
			if ref != key {
				keys = append(keys, ref)
			}
		}
		if len(keys) > 0 {
			criterion.EvidenceLocalKeys = keys
			criteria = append(criteria, criterion)
		}
	}
	material.CriterionLocalEvidence = criteria
	return nil
}

// Large review contracts contain authenticated references. Expand the original
// specification outside local locks, validate the digest/target, and use it only
// as a derived view; the server-issued review document remains byte-identical.
func materialReviewViewV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, id d.ID, kind string) (*ContractViewV2, error) {
	if kind != "review" {
		return nil, nil
	}
	_, view, _, _, e := loadContractV2(w, profile, id)
	if e != nil {
		return nil, e
	}
	if view.ReviewSpecification == nil {
		return nil, usage("review contract required")
	}
	if view.ReviewSpecification.Spec.IssuedWorkSpecification != nil {
		return nil, nil
	}
	view, e = expandReviewSpecificationV2(ctx, profile, client, view)
	if e != nil {
		return nil, e
	}
	return &view, nil
}

func expandReviewSpecificationV2(ctx context.Context, profile ProfileV2, client *sdk.Client, view ContractViewV2) (ContractViewV2, error) {
	if view.ReviewSpecification == nil || view.ReviewSpecification.Spec.IssuedWorkSpecification != nil {
		return view, nil
	}
	if client == nil {
		return view, fmt.Errorf("original work specification requires the authenticated server connection")
	}
	for _, ref := range view.ReviewSpecification.Spec.Omitted {
		if ref.Kind != "execution_specification" {
			continue
		}
		if ref.Digest != view.ReviewSpecification.Spec.IssuedWorkSpecDigest || ref.ContractID.Validate() != nil {
			return view, fmt.Errorf("reviewed specification reference differs")
		}
		scope := d.Scope{TenantID: profile.Binding.TenantID, OutcomeID: d.ID(view.Authority.OutcomeID)}
		state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "specification", ContractKind: "execution", ID: ref.ContractID, Digest: ref.Digest})
		if e != nil {
			return view, e
		}
		if state.Envelope == nil {
			return view, fmt.Errorf("referenced reviewed specification unavailable")
		}
		document, e := signing.ToDocument(*state.Envelope)
		if e != nil {
			return view, e
		}
		raw, e := verifyProfileDocumentV2(profile, document, signing.ContractSpec)
		if e != nil {
			return view, e
		}
		var spec signing.SpecPayload[d.SignedWorkSpec]
		if e = signing.DecodeHistoricalPayload(raw, &spec, 128<<10); e != nil {
			return view, e
		}
		if signing.Digest(raw) != ref.Digest || spec.ContractID != ref.ContractID.String() || spec.ServerID != view.Authority.ServerID || spec.TenantID != view.Authority.TenantID || spec.OutcomeID != view.Authority.OutcomeID || spec.WorkItemID != view.Authority.WorkItemID || spec.ContractKind != "execution" {
			return view, fmt.Errorf("referenced reviewed specification target/digest differs")
		}
		copy := *view.ReviewSpecification
		copy.Spec.IssuedWorkSpecification = &document
		view.ReviewSpecification = &copy
		return view, nil
	}
	return view, fmt.Errorf("review contract has no original specification or authenticated expansion reference")
}
