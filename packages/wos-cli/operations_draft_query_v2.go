package woscli

import (
	"strconv"
	"unicode/utf8"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Continuation shows staged decisions and references, with explicit expansion
// paths for bounded lists. It never implies that a draft has been accepted.
func compactDraftV2(file ContractFileV2) map[string]any {
	omitted := map[string]int{}
	clip := func(value, field string, maximum int) string {
		if len(value) <= maximum {
			return value
		}
		end := maximum
		for end > 0 && !utf8.ValidString(value[:end]) {
			end--
		}
		omitted[field+"_bytes"] += len(value) - end
		return value[:end]
	}
	ids := func(values []d.ID, field string) []d.ID {
		if len(values) > 5 {
			omitted[field] += len(values) - 5
			return values[:5]
		}
		return values
	}
	keys := func(values []string, field string) []string {
		if len(values) > 5 {
			omitted[field] += len(values) - 5
			return values[:5]
		}
		return values
	}
	data := map[string]any{"local_staged": file.Local.AcceptanceReceipt == nil, "remote_committed": file.Local.AcceptanceReceipt != nil, "omitted": omitted}
	var evidence []a.SyncEvidenceInput
	var artifacts []a.SyncArtifactInput
	assessments := []any{}
	for i, value := range file.draftAssessmentsV2() {
		if i >= 5 {
			omitted["assessments"]++
			continue
		}
		assessments = append(assessments, map[string]any{"criterion_id": value.CriterionID, "criterion_revision": strconv.FormatUint(uint64(value.CriterionRevision), 10), "result": value.Result, "rationale": clip(value.Rationale, "assessment.rationale", 1024), "evidence_ids": ids(value.EvidenceIDs, "assessment.evidence_ids"), "evidence_local_keys": keys(value.EvidenceLocalKeys, "assessment.evidence_local_keys")})
	}
	data["assessments"] = assessments
	if file.Execution != nil {
		material := file.Execution.Material
		data["summary"] = clip(material.Result.Summary, "summary", 2048)
		data["evidence_ids"] = ids(material.Result.EvidenceIDs, "evidence_ids")
		references := []any{}
		for i, value := range material.Result.Artifacts {
			if i >= 5 {
				omitted["artifact_refs"]++
				continue
			}
			references = append(references, map[string]any{"artifact_id": value.ArtifactID, "source_version": clip(value.SourceVersion, "artifact.source_version", 512), "checksum": clip(value.Checksum, "artifact.checksum", 256)})
		}
		data["artifact_refs"] = references
		criteria := material.Result.CriterionEvidence
		if len(criteria) > 5 {
			omitted["criterion_evidence"] = len(criteria) - 5
			criteria = criteria[:5]
		}
		criterionViews := []any{}
		for _, criterion := range criteria {
			criterionViews = append(criterionViews, map[string]any{"criterion_id": criterion.CriterionID, "criterion_revision": strconv.FormatUint(uint64(criterion.CriterionRevision), 10), "evidence_ids": ids(criterion.EvidenceIDs, "criterion_evidence.evidence_ids")})
		}
		data["criterion_evidence"] = criterionViews
		responses := []any{}
		for i, value := range material.CorrectionResponses {
			if i >= 5 {
				omitted["correction_responses"]++
				continue
			}
			responses = append(responses, map[string]any{"finding_id": value.FindingID, "summary": clip(value.Summary, "correction.summary", 1024), "evidence_ids": ids(value.EvidenceIDs, "correction.evidence_ids")})
		}
		data["correction_responses"] = responses
		evidence, artifacts = material.Evidence, material.Artifacts
	} else {
		material := file.Review.Material
		data["decision"], data["reason"] = file.Review.Decision, clip(material.Reason, "reason", 2048)
		findings := []any{}
		for i, value := range material.Findings {
			if i >= 5 {
				omitted["findings"]++
				continue
			}
			findings = append(findings, map[string]any{"id": value.ID, "criterion_id": value.CriterionID, "requirement_ref": value.RequirementRef, "description": clip(value.Description, "finding.description", 1024)})
		}
		data["findings"] = findings
		evidence, artifacts = material.Evidence, material.Artifacts
	}
	evidenceViews := []any{}
	for i, value := range evidence {
		if i >= 5 {
			omitted["staged_evidence"]++
			continue
		}
		evidenceViews = append(evidenceViews, map[string]any{"local_key": value.LocalKey, "type": value.Evidence.EvidenceType, "description": clip(value.Evidence.Description, "evidence.description", 512), "uri": clip(value.Evidence.SourceRef.URI, "evidence.uri", 1024), "source_version": clip(value.Evidence.SourceVersion, "evidence.source_version", 512), "checksum": clip(value.Evidence.Checksum, "evidence.checksum", 256)})
	}
	artifactViews := []any{}
	for i, value := range artifacts {
		if i >= 5 {
			omitted["staged_artifacts"]++
			continue
		}
		artifactViews = append(artifactViews, map[string]any{"local_key": value.LocalKey, "name": clip(value.Artifact.Name, "artifact.name", 512), "uri": clip(value.Artifact.URI, "artifact.uri", 1024), "source_version": clip(value.Artifact.SourceVersion, "artifact.source_version", 512), "checksum": clip(value.Artifact.Checksum, "artifact.checksum", 256)})
	}
	data["evidence"], data["artifacts"] = evidenceViews, artifactViews
	data["expand"] = "Use assessment/finding/correction list; review evidence/artifact list expands staged observations. Work evidence/artifact list reads registered server records."
	return data
}

func compactRecentActivityV2(page a.WorkActivityPage) map[string]any {
	items := []any{}
	for _, activity := range page.Items {
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
		refs := func(values []d.ID, field string) []d.ID {
			if len(values) > 5 {
				omitted[field] = len(values) - 5
				return values[:5]
			}
			return values
		}
		versions := []any{}
		for i, value := range activity.ArtifactVersions {
			if i >= 5 {
				omitted["artifact_versions"]++
				continue
			}
			versions = append(versions, map[string]any{"artifact_id": value.ArtifactID, "source_version": clip(value.SourceVersion, "artifact.source_version", 512), "checksum": clip(value.Checksum, "artifact.checksum", 256)})
		}
		criteria := activity.CriterionRefs
		if len(criteria) > 5 {
			omitted["criterion_refs"] = len(criteria) - 5
			criteria = criteria[:5]
		}
		items = append(items, map[string]any{"id": activity.ID, "contract_id": activity.ContractID, "work_item_id": activity.WorkItemID, "type": activity.Type, "description": clip(activity.Description, "description", 1024), "sequence": strconv.FormatUint(activity.Sequence, 10), "principal_id": activity.PrincipalID, "recorded_at": activity.RecordedAt, "artifact_refs": refs(activity.ArtifactRefs, "artifact_refs"), "evidence_refs": refs(activity.EvidenceRefs, "evidence_refs"), "criterion_refs": criteria, "artifact_versions": versions, "omitted": omitted})
	}
	return map[string]any{"items": items, "next_cursor": page.NextCursor, "order": page.Order, "outcome_revision": strconv.FormatUint(uint64(page.OutcomeRevision), 10), "history_through_revision": strconv.FormatUint(uint64(page.HistoryThroughRevision), 10), "expand": "Use work history with the same WorkItem, Outcome and descending order for complete records."}
}
