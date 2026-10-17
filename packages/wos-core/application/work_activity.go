package application

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// AppendWorkActivityCommand records a declaration under current signed authority.
// Unlike return, it neither signs an execution result nor concludes a Task.
type AppendWorkActivityCommand struct {
	Scope                   d.Scope
	ContractID              d.ID
	SignerKeyID             d.ID
	Authority               ContractAuthority
	AuthorityDigest         string
	ExpectedContractVersion d.Version `json:"expected_contract_version,string"`
	Type                    d.WorkActivityType
	Description             string                       `wos:"required"`
	ArtifactRefs            []d.ID                       `wos:"optional"`
	EvidenceRefs            []d.ID                       `wos:"optional"`
	CriterionRefs           []d.WorkActivityCriterionRef `wos:"optional"`
	Artifacts               []SyncArtifactInput          `wos:"optional"`
	Evidence                []SyncEvidenceInput          `wos:"optional"`
	EvidenceLinks           []SyncEvidenceLinkInput      `wos:"optional"`
}

type WorkActivityResult struct {
	Activity      d.WorkActivity   `json:"activity"`
	Artifacts     []d.Artifact     `json:"artifacts,omitempty"`
	Evidence      []d.Evidence     `json:"evidence,omitempty"`
	EvidenceLinks []d.EvidenceLink `json:"evidence_links,omitempty"`
	LocalKeys     map[string]d.ID  `json:"local_keys,omitempty"`
}

func (s *Service) AppendWorkActivity(ctx context.Context, cc d.CommandContext, cmd AppendWorkActivityCommand) (MutationResult[WorkActivityResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkActivityResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[WorkActivityResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "activity requires a durable idempotency intent")
	}
	if err := boundedContractInput(cmd); err != nil {
		return MutationResult[WorkActivityResult]{}, err
	}
	if !cmd.Type.Valid() || strings.TrimSpace(cmd.Description) == "" || len(cmd.Description) > 16384 {
		return MutationResult[WorkActivityResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "activity requires type and description up to 16 KiB")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (WorkActivityResult, d.OutcomeRevision, error) {
		var zero WorkActivityResult
		if err := cmd.SignerKeyID.Validate(); err != nil {
			return zero, 0, err
		}
		identity, _, err := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionWorkWrite, cmd.SignerKeyID)
		if err != nil {
			return zero, 0, err
		}
		if _, err = s.requireSignedIssuer(ctx, u, cmd.Scope.TenantID); err != nil {
			return zero, 0, err
		}
		if _, err = u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		if err = requireActiveOutcome(ctx, u, cmd.Scope); err != nil {
			return zero, 0, err
		}
		_, contracts, err := signedRepository(u)
		if err != nil {
			return zero, 0, err
		}
		contract, err := contracts.Get(ctx, cmd.Scope, cmd.ContractID)
		if err != nil {
			return zero, 0, err
		}
		if contract.SignedBinding == nil || contract.SignedBinding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "activity belongs to another credential/Principal")
		}
		permittedKey := false
		for _, key := range contract.SignedBinding.AllowedSigningKeyIDs {
			if key == cmd.SignerKeyID {
				permittedKey = true
			}
		}
		if !permittedKey {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "activity signing key is outside issued authority")
		}
		if err = requireIssuedWorkAuthority(ctx, u, contract, cmd.Authority, cmd.AuthorityDigest); err != nil {
			return zero, 0, err
		}
		if contract.Version != cmd.ExpectedContractVersion {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "activity contract version differs")
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		if err = contract.Authorize(identity.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, now); err != nil {
			return zero, 0, err
		}
		work, err := u.WorkItems().Get(ctx, cmd.Scope, contract.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		if work.CurrentContractID == nil || *work.CurrentContractID != contract.ID {
			return zero, 0, d.NewError(d.ErrorCodeStaleExecution, "activity contract no longer owns Task")
		}
		// Inline observations may describe this Task and its frozen criteria only.
		for _, link := range cmd.EvidenceLinks {
			if link.Link.TargetRef != work.Ref() {
				return zero, 0, d.NewError(d.ErrorCodeInvalidScope, "activity evidence must target its contracted Task")
			}
			if link.Link.CriterionID != nil {
				found := false
				for _, criterion := range contract.Spec.Criteria {
					if criterion.ID == *link.Link.CriterionID && criterion.Status == d.CriterionStatusActive {
						found = true
					}
				}
				if !found {
					return zero, 0, d.NewError(d.ErrorCodeCriterion, "activity evidence criterion is outside frozen specification")
				}
			}
		}
		registered, err := s.registerContractRecords(ctx, u, cc, SyncWorkContractCommand{Scope: cmd.Scope, Artifacts: cmd.Artifacts, Evidence: cmd.Evidence, EvidenceLinks: cmd.EvidenceLinks}, now)
		if err != nil {
			return zero, 0, err
		}
		activityID, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		activity := d.WorkActivity{ID: activityID, Scope: cmd.Scope, ContractID: contract.ID, WorkItemID: work.ID, Type: cmd.Type, Description: strings.TrimSpace(cmd.Description), PrincipalID: identity.PrincipalID, Actor: cc.Actor, RecordedAt: now, ArtifactRefs: append([]d.ID{}, cmd.ArtifactRefs...), EvidenceRefs: append([]d.ID{}, cmd.EvidenceRefs...), CriterionRefs: append([]d.WorkActivityCriterionRef{}, cmd.CriterionRefs...)}
		for _, a := range registered.Artifacts {
			activity.ArtifactRefs = append(activity.ArtifactRefs, a.ID)
		}
		for _, e := range registered.Evidence {
			activity.EvidenceRefs = append(activity.EvidenceRefs, e.ID)
		}
		// Preserve the activity-to-evidence relationship even when a link uses an
		// already registered observation rather than creating another copy.
		for _, link := range registered.EvidenceLinks {
			found := false
			for _, id := range activity.EvidenceRefs {
				if id == link.EvidenceID {
					found = true
				}
			}
			if !found {
				activity.EvidenceRefs = append(activity.EvidenceRefs, link.EvidenceID)
			}
			if link.CriterionID != nil {
				found = false
				for _, ref := range activity.CriterionRefs {
					if ref.CriterionID == *link.CriterionID {
						found = true
					}
				}
				if !found {
					for _, c := range contract.Spec.Criteria {
						if c.ID == *link.CriterionID {
							activity.CriterionRefs = append(activity.CriterionRefs, d.WorkActivityCriterionRef{CriterionID: c.ID, CriterionRevision: c.Revision})
						}
					}
				}
			}
		}
		artifacts, evidence, _, _, err := documentaryRepositories(u)
		if err != nil {
			return zero, 0, err
		}
		for _, id := range activity.ArtifactRefs {
			a, err := artifacts.Get(ctx, cmd.Scope, id)
			if err != nil {
				return zero, 0, err
			}
			if a.Lifecycle != d.ArtifactLifecycleRegistered {
				return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "activity artifact was withdrawn")
			}
			activity.ArtifactVersions = append(activity.ArtifactVersions, d.SubmissionArtifact{ArtifactID: a.ID, SourceVersion: a.SourceVersion, Checksum: a.Checksum})
		}
		for _, id := range activity.EvidenceRefs {
			e, err := evidence.Get(ctx, cmd.Scope, id)
			if err != nil {
				return zero, 0, err
			}
			if e.Lifecycle != d.EvidenceLifecycleRegistered {
				return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "activity evidence was retracted")
			}
		}
		for _, ref := range activity.CriterionRefs {
			found := false
			for _, c := range contract.Spec.Criteria {
				if c.ID == ref.CriterionID && c.Revision == ref.CriterionRevision && c.Status == d.CriterionStatusActive {
					found = true
				}
			}
			if !found {
				return zero, 0, d.NewError(d.ErrorCodeCriterion, "activity criterion revision is outside frozen specification")
			}
		}
		store, ok := u.(ports.WorkActivityUnitOfWork)
		if !ok {
			return zero, 0, d.NewError(d.ErrorCodeInvalidConfig, "durable activity repository unavailable")
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return zero, 0, err
		}
		activity.Sequence = uint64(rev)
		if err = store.WorkActivities().Insert(ctx, activity); err != nil {
			return zero, 0, err
		}
		return WorkActivityResult{Activity: activity, Artifacts: registered.Artifacts, Evidence: registered.Evidence, EvidenceLinks: registered.EvidenceLinks, LocalKeys: registered.LocalKeys}, rev, nil
	})
}

type WorkActivityPage struct {
	Order                  string            `json:"order"`
	Items                  []d.WorkActivity  `json:"items"`
	NextCursor             string            `json:"next_cursor,omitempty"`
	OutcomeRevision        d.OutcomeRevision `json:"outcome_revision,string"`
	HistoryThroughRevision d.OutcomeRevision `json:"history_through_revision,string"`
}

func workActivityEvents(s *Service, cc d.CommandContext, meta commandMetadata, result WorkActivityResult, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	activity := result.Activity
	ref := d.EntityRef{Scope: activity.Scope, Kind: d.EntityKindWorkItem, ID: activity.WorkItemID}
	specs := []compoundEventSpec{{eventType: "work_activity.recorded", ref: ref}}
	for _, a := range result.Artifacts {
		specs = append(specs, compoundEventSpec{eventType: "artifact.registered", ref: a.Ref(), after: versionPtr(a.Version)})
	}
	for _, e := range result.Evidence {
		specs = append(specs, compoundEventSpec{eventType: "evidence.registered", ref: e.Ref(), after: versionPtr(e.Version)})
	}
	for _, link := range result.EvidenceLinks {
		specs = append(specs, compoundEventSpec{eventType: "evidence.link_created", ref: link.Ref(), after: versionPtr(link.Version)})
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, err
	}
	for i := range events {
		payload := any(map[string]any{"activity_id": activity.ID, "contract_id": activity.ContractID, "work_item_id": activity.WorkItemID})
		if i == 0 {
			payload = activity
		}
		events[i].Payload, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		events[i].RecordedAt = activity.RecordedAt
		if err = events[i].Validate(); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) ListWorkActivity(ctx context.Context, scope d.Scope, filter ports.WorkActivityFilter, limit int, cursor string) (WorkActivityPage, error) {
	var zero WorkActivityPage
	if err := scope.Validate(); err != nil {
		return zero, err
	}
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return zero, err
	}
	for _, id := range []d.ID{filter.WorkItemID, filter.ContractID} {
		if !id.IsZero() {
			if err := id.Validate(); err != nil {
				return zero, err
			}
		}
	}
	n, err := queryLimit(limit)
	if err != nil {
		return zero, err
	}
	// Callers select identity filters only; pagination bounds are server-owned.
	filter.AfterSequence, filter.ThroughSequence, filter.Limit = 0, 0, 0
	fingerprint := filterHash(filter)
	var decoded queryCursor
	if cursor != "" {
		decoded, err = decodeCursor(cursor, scope.TenantID, scope.OutcomeID, "work_activity", fingerprint)
		if err != nil {
			return zero, err
		}
		filter.AfterSequence, err = strconv.ParseUint(decoded.Key, 10, 64)
		if err != nil || decoded.Revision == 0 || filter.AfterSequence > uint64(decoded.Revision) {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: activity chronology")
		}
	}
	u, err := s.beginReadSnapshot(ctx)
	if err != nil {
		return zero, err
	}
	defer u.Rollback()
	if err = s.authorizeScopedReadInUnitOfWork(ctx, u, scope); err != nil {
		return zero, err
	}
	coord, err := u.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return zero, err
	}
	filter.ThroughSequence = uint64(coord.Revision)
	if cursor != "" {
		if decoded.Revision > coord.Revision {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: activity revision is in the future")
		}
		filter.ThroughSequence = uint64(decoded.Revision)
	}
	order := "asc"
	if filter.Descending {
		order = "desc"
	}
	page := WorkActivityPage{Order: order, Items: []d.WorkActivity{}, OutcomeRevision: coord.Revision, HistoryThroughRevision: d.OutcomeRevision(filter.ThroughSequence)}
	if !filter.ContractID.IsZero() {
		_, contracts, err := signedRepository(u)
		if err != nil {
			return zero, err
		}
		contract, err := contracts.Get(ctx, scope, filter.ContractID)
		if err != nil {
			return zero, err
		}
		if !filter.WorkItemID.IsZero() && filter.WorkItemID != contract.WorkItemID {
			return zero, d.NewError(d.ErrorCodeInvalidScope, "activity contract does not belong to selected Task")
		}
	}
	if filter.ThroughSequence == 0 {
		return page, nil
	}
	store, ok := u.(ports.WorkActivityUnitOfWork)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "durable activity repository unavailable")
	}
	filter.Limit = n + 1
	page.Items, err = store.WorkActivities().List(ctx, scope, filter)
	if err != nil {
		return zero, err
	}
	more := len(page.Items) > n
	if more {
		page.Items = page.Items[:n]
	}
	for {
		if more && len(page.Items) > 0 {
			page.NextCursor = encodeCursor(queryCursor{Tenant: scope.TenantID, Outcome: scope.OutcomeID, Section: "work_activity", Filter: fingerprint, Revision: page.HistoryThroughRevision, Key: strconv.FormatUint(page.Items[len(page.Items)-1].Sequence, 10)})
		}
		raw, err := json.Marshal(page)
		if err != nil {
			return zero, err
		}
		if len(raw) <= MaxSnapshotBytes {
			return page, nil
		}
		if len(page.Items) <= 1 {
			return zero, d.NewError(d.ErrorCodeGraphLimitExceeded, "activity record exceeds byte limit")
		}
		page.Items = page.Items[:len(page.Items)-1]
		more = true
	}
}
