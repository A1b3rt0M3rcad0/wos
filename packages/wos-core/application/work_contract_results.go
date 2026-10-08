package application

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type ContractCheckpointInput struct {
	Summary       string
	Completed     []string    `wos:"optional"`
	Pending       []string    `wos:"optional"`
	NextAction    string      `wos:"optional"`
	ArtifactRefs  []domain.ID `wos:"optional"`
	EvidenceRefs  []domain.ID `wos:"optional"`
	IssueRefs     []domain.ID `wos:"optional"`
	RepositoryRef string      `wos:"optional"`
	WorkingCommit string      `wos:"optional"`
	Dirty         bool        `wos:"optional"`
	Unknown       []string    `wos:"optional"`
}
type SyncArtifactInput struct {
	LocalKey string
	Artifact RegisterArtifactCommand
}
type SyncEvidenceInput struct {
	LocalKey         string
	ArtifactLocalKey string `wos:"optional"`
	Evidence         RegisterEvidenceCommand
}
type SyncEvidenceLinkInput struct {
	LocalKey         string
	EvidenceLocalKey string `wos:"optional"`
	Link             CreateEvidenceLinkCommand
}
type SyncWorkContractCommand struct {
	Artifacts               []SyncArtifactInput     `wos:"optional"`
	Evidence                []SyncEvidenceInput     `wos:"optional"`
	EvidenceLinks           []SyncEvidenceLinkInput `wos:"optional"`
	Scope                   domain.Scope
	ContractID              domain.ID
	Authority               ContractAuthority
	ExpectedContractVersion domain.Version
	Checkpoint              ContractCheckpointInput
}
type SubmitWorkResultCommand struct {
	Scope                   domain.Scope
	ContractID              domain.ID
	Authority               ContractAuthority
	ExpectedContractVersion domain.Version
	Material                domain.WorkResultMaterial
	SupersedesSubmissionID  *domain.ID `wos:"optional"`
}
type FinalizeWorkContractCommand struct {
	Scope                   domain.Scope
	ContractID              domain.ID
	Authority               ContractAuthority
	ExpectedContractVersion domain.Version
	ExpectedWorkItemVersion domain.Version
	SubmissionID            domain.ID
	Reason                  string
}

func (s *Service) contractForWrite(ctx context.Context, u ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, id domain.ID, authority ContractAuthority) (ports.WorkContractRepository, domain.WorkContract, domain.WorkItem, time.Time, error) {
	var c domain.WorkContract
	var w domain.WorkItem
	var now time.Time
	if err := requireActiveOutcome(ctx, u, scope); err != nil {
		return nil, c, w, now, err
	}
	repo, err := contractRepository(u)
	if err != nil {
		return nil, c, w, now, err
	}
	c, err = repo.Get(ctx, scope, id)
	if err != nil {
		return nil, c, w, now, err
	}
	if err = validateContractSpec(c, authority); err != nil {
		return nil, c, w, now, err
	}
	now, err = s.transactionTime(ctx, u)
	if err != nil {
		return nil, c, w, now, err
	}
	if err = c.Authorize(cc.PrincipalID, authority.ExecutionID, authority.FencingToken, now); err != nil {
		return nil, c, w, now, err
	}
	w, err = u.WorkItems().Get(ctx, scope, c.WorkItemID)
	if err != nil {
		return nil, c, w, now, err
	}
	if w.CurrentContractID == nil || *w.CurrentContractID != c.ID {
		return nil, c, w, now, domain.NewError(domain.ErrorCodeStaleExecution, "contract no longer owns work")
	}
	return repo, c, w, now, nil
}
func boundedContractInput(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(b) > 256*1024 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "contract input exceeds 256 KiB")
	}
	return nil
}
func (s *Service) SyncWorkContract(ctx context.Context, cc domain.CommandContext, cmd SyncWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if err := boundedContractInput(cmd); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (WorkContractResult, domain.OutcomeRevision, error) {
		zero := WorkContractResult{}
		repo, c, w, now, err := s.contractForWrite(ctx, u, cc, cmd.Scope, cmd.ContractID, cmd.Authority)
		if err != nil {
			return zero, 0, err
		}
		artifacts, evidence, links, decisions, err := documentaryRepositories(u)
		if err != nil {
			return zero, 0, err
		}
		if len(cmd.Artifacts)+len(cmd.Evidence)+len(cmd.EvidenceLinks) > 100 {
			return zero, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "sync record limit is 100")
		}
		mappings := map[string]domain.ID{}
		nextID := func(key string) (domain.ID, error) {
			if len(key) == 0 || len(key) > 128 {
				return "", domain.NewError(domain.ErrorCodeInvalidArgument, "local_key is required and limited to 128 bytes")
			}
			if _, ok := mappings[key]; ok {
				return "", domain.NewError(domain.ErrorCodeInvalidArgument, "duplicate local_key")
			}
			id, err := s.ids.NewID()
			if err == nil {
				mappings[key] = id
			}
			return id, err
		}
		createdArtifacts := []domain.Artifact{}
		createdEvidence := []domain.Evidence{}
		createdLinks := []domain.EvidenceLink{}
		for _, entry := range cmd.Artifacts {
			id, err := nextID(entry.LocalKey)
			if err != nil {
				return zero, 0, err
			}
			x := entry.Artifact
			if x.Scope != cmd.Scope {
				return zero, 0, domain.NewError(domain.ErrorCodeInvalidScope, "artifact scope must match sync")
			}
			value, err := domain.NewArtifact(id, cmd.Scope, x.ArtifactType, x.Name, x.URI, x.MediaType, x.Checksum, x.SourceVersion, cc.Actor, x.ProducedAt, now)
			if err != nil {
				return zero, 0, err
			}
			if err = artifacts.Insert(ctx, value); err != nil {
				return zero, 0, err
			}
			createdArtifacts = append(createdArtifacts, value)
			cmd.Checkpoint.ArtifactRefs = append(cmd.Checkpoint.ArtifactRefs, id)
		}
		for _, entry := range cmd.Evidence {
			id, err := nextID(entry.LocalKey)
			if err != nil {
				return zero, 0, err
			}
			x := entry.Evidence
			if x.Scope != cmd.Scope {
				return zero, 0, domain.NewError(domain.ErrorCodeInvalidScope, "evidence scope must match sync")
			}
			if entry.ArtifactLocalKey != "" {
				aid, ok := mappings[entry.ArtifactLocalKey]
				if !ok {
					return zero, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown artifact local_key")
				}
				x.ArtifactID = &aid
			}
			if x.ArtifactID != nil {
				a, err := artifacts.Get(ctx, cmd.Scope, *x.ArtifactID)
				if err != nil {
					return zero, 0, err
				}
				if a.Lifecycle != domain.ArtifactLifecycleRegistered {
					return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "artifact withdrawn")
				}
			}
			value, err := domain.NewEvidence(id, cmd.Scope, x.EvidenceType, x.Description, x.SourceRef, cc.Actor, x.CapturedAt, x.ArtifactID, x.Measurement, x.SourceVersion, x.Checksum, now)
			if err != nil {
				return zero, 0, err
			}
			if err = evidence.Insert(ctx, value); err != nil {
				return zero, 0, err
			}
			createdEvidence = append(createdEvidence, value)
			cmd.Checkpoint.EvidenceRefs = append(cmd.Checkpoint.EvidenceRefs, id)
		}
		for _, entry := range cmd.EvidenceLinks {
			id, err := nextID(entry.LocalKey)
			if err != nil {
				return zero, 0, err
			}
			x := entry.Link
			if x.Scope != cmd.Scope || x.TargetRef.Scope != cmd.Scope {
				return zero, 0, domain.NewError(domain.ErrorCodeInvalidScope, "link scope must match sync")
			}
			if entry.EvidenceLocalKey != "" {
				eid, ok := mappings[entry.EvidenceLocalKey]
				if !ok {
					return zero, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown evidence local_key")
				}
				x.EvidenceID = eid
			}
			ev, err := evidence.Get(ctx, cmd.Scope, x.EvidenceID)
			if err != nil {
				return zero, 0, err
			}
			if ev.Lifecycle != domain.EvidenceLifecycleRegistered {
				return zero, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "evidence retracted")
			}
			if err = validateEvidenceLinkTarget(ctx, u, decisions, x.TargetRef, x.CriterionID); err != nil {
				return zero, 0, err
			}
			value, err := domain.NewEvidenceLink(id, cmd.Scope, x.EvidenceID, x.TargetRef, x.CriterionID, x.Stance, x.Rationale, now)
			if err != nil {
				return zero, 0, err
			}
			if err = links.Insert(ctx, value); err != nil {
				return zero, 0, err
			}
			createdLinks = append(createdLinks, value)
		}
		for _, id := range cmd.Checkpoint.ArtifactRefs {
			if _, err = artifacts.Get(ctx, cmd.Scope, id); err != nil {
				return zero, 0, err
			}
		}
		for _, id := range cmd.Checkpoint.EvidenceRefs {
			if _, err = evidence.Get(ctx, cmd.Scope, id); err != nil {
				return zero, 0, err
			}
		}
		issues, _, err := issueBlockerRepositories(u)
		if err != nil {
			return zero, 0, err
		}
		for _, id := range cmd.Checkpoint.IssueRefs {
			if _, err = issues.Get(ctx, cmd.Scope, id); err != nil {
				return zero, 0, err
			}
		}
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		p := domain.WorkCheckpoint{ID: id, Scope: cmd.Scope, ContractID: c.ID, WorkItemID: w.ID, Sequence: uint64(c.Version), PrincipalID: cc.PrincipalID, AcceptedAt: now, Summary: cmd.Checkpoint.Summary, Completed: cmd.Checkpoint.Completed, Pending: cmd.Checkpoint.Pending, NextAction: cmd.Checkpoint.NextAction, ArtifactRefs: cmd.Checkpoint.ArtifactRefs, EvidenceRefs: cmd.Checkpoint.EvidenceRefs, IssueRefs: cmd.Checkpoint.IssueRefs, RepositoryRef: cmd.Checkpoint.RepositoryRef, WorkingCommit: cmd.Checkpoint.WorkingCommit, Dirty: cmd.Checkpoint.Dirty, Unknown: cmd.Checkpoint.Unknown}
		v, l := c.Version, c.LeaseVersion
		if err = c.RecordCheckpoint(p, cc.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedContractVersion, now); err != nil {
			return zero, 0, err
		}
		if err = repo.InsertCheckpoint(ctx, p); err != nil {
			return zero, 0, err
		}
		if err = repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{Contract: c, WorkItem: w, EvaluatedAt: now, Checkpoint: &p, LocalKeys: mappings, Artifacts: createdArtifacts, Evidence: createdEvidence, EvidenceLinks: createdLinks}, rev, err
	})
}
func validateSubmittedMaterial(ctx context.Context, u ports.UnitOfWork, w domain.WorkItem, m domain.WorkResultMaterial) error {
	artifacts, _, _, _, err := documentaryRepositories(u)
	if err != nil {
		return err
	}
	seen := map[domain.ID]bool{}
	for _, ref := range m.Artifacts {
		if seen[ref.ArtifactID] {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "duplicate submitted artifact")
		}
		seen[ref.ArtifactID] = true
		a, err := artifacts.Get(ctx, w.Scope, ref.ArtifactID)
		if err != nil {
			return err
		}
		if a.Lifecycle != domain.ArtifactLifecycleRegistered || (ref.SourceVersion != "" && ref.SourceVersion != a.SourceVersion) || (ref.Checksum != "" && ref.Checksum != a.Checksum) {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "artifact is withdrawn or material revision differs")
		}
	}
	if err = validateAssessmentEvidence(ctx, u, w.Scope, m.EvidenceIDs); err != nil {
		return err
	}
	evidence := map[domain.ID]bool{}
	for _, id := range m.EvidenceIDs {
		evidence[id] = true
	}
	criteria := map[domain.ID]bool{}
	for _, ref := range m.CriterionEvidence {
		if criteria[ref.CriterionID] {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "duplicate submitted criterion")
		}
		criteria[ref.CriterionID] = true
		found := false
		for _, c := range w.Criteria.Items {
			if c.ID == ref.CriterionID && c.Revision == ref.CriterionRevision && c.Status == domain.CriterionStatusActive {
				found = true
			}
		}
		if !found {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion revision is not current")
		}
		for _, id := range ref.EvidenceIDs {
			if !evidence[id] {
				return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion evidence is absent from submitted material")
			}
		}
	}
	return nil
}
func (s *Service) SubmitWorkResult(ctx context.Context, cc domain.CommandContext, cmd SubmitWorkResultCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if err := boundedContractInput(cmd); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (WorkContractResult, domain.OutcomeRevision, error) {
		zero := WorkContractResult{}
		repo, c, w, now, err := s.contractForWrite(ctx, u, cc, cmd.Scope, cmd.ContractID, cmd.Authority)
		if err != nil {
			return zero, 0, err
		}
		if err = validateSubmittedMaterial(ctx, u, w, cmd.Material); err != nil {
			return zero, 0, err
		}
		material := domain.NormalizeResultMaterial(cmd.Material)
		digest, err := domain.SemanticDigest(material)
		if err != nil {
			return zero, 0, err
		}
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		p := domain.WorkSubmission{ID: id, Scope: cmd.Scope, Material: material, Digest: digest, PrincipalID: cc.PrincipalID, Actor: cc.Actor, SubmittedAt: now, SupersedesSubmissionID: cmd.SupersedesSubmissionID}
		v, l := c.Version, c.LeaseVersion
		if err = c.Submit(p, cc.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedContractVersion, now); err != nil {
			return zero, 0, err
		}
		if err = repo.InsertSubmission(ctx, p); err != nil {
			return zero, 0, err
		}
		if err = repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{Contract: c, WorkItem: w, EvaluatedAt: now, Submission: &p}, rev, err
	})
}
func (s *Service) FinalizeWorkContract(ctx context.Context, cc domain.CommandContext, cmd FinalizeWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (WorkContractResult, domain.OutcomeRevision, error) {
		zero := WorkContractResult{}
		repo, c, w, now, err := s.contractForWrite(ctx, u, cc, cmd.Scope, cmd.ContractID, cmd.Authority)
		if err != nil {
			return zero, 0, err
		}
		candidate := w
		candidate.Lifecycle = domain.WorkItemLifecycleTodo
		candidate.CurrentLease = nil
		if err = requireWorkItemReady(ctx, u, candidate, now); err != nil {
			return zero, 0, err
		}
		if err = requireNotBlocked(ctx, u, w.Ref()); err != nil {
			return zero, 0, err
		}
		if err = requireHardDependenciesSatisfied(ctx, u, w.Ref()); err != nil {
			return zero, 0, err
		}
		p, err := repo.GetSubmission(ctx, cmd.Scope, cmd.SubmissionID)
		if err != nil {
			return zero, 0, err
		}
		if err = validateSubmittedMaterial(ctx, u, w, p.Material); err != nil {
			return zero, 0, err
		}
		for _, assessment := range w.Criteria.CurrentAssessments {
			if err = validateAssessmentEvidence(ctx, u, w.Scope, assessment.EvidenceIDs); err != nil {
				return zero, 0, err
			}
		}
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		conclusion := conclusionFromContext(id, cc, cmd.Reason, now)
		v, l, wv := c.Version, c.LeaseVersion, w.Version
		if err = c.Finalize(&w, p, cc.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedContractVersion, cmd.ExpectedWorkItemVersion, conclusion, now); err != nil {
			return zero, 0, err
		}
		if err = repo.Save(ctx, c, v, l); err != nil {
			return zero, 0, err
		}
		if err = u.WorkItems().Save(ctx, w, wv); err != nil {
			return zero, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return WorkContractResult{Contract: c, WorkItem: w, EvaluatedAt: now, Submission: &p}, rev, err
	})
}

func bindSubmissionAssessment(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef, id *domain.ID, a *domain.CriterionAssessment) error {
	if owner.Kind != domain.EntityKindWorkItem {
		if id != nil {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "submission assessment requires work owner")
		}
		return nil
	}
	w, err := u.WorkItems().Get(ctx, owner.Scope, owner.ID)
	if err != nil {
		return err
	}
	if !w.ContractsEnabled {
		if id != nil {
			return domain.NewError(domain.ErrorCodeContractProtocolRequired, "legacy work cannot bind submission")
		}
		return nil
	}
	if id == nil {
		return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "contract work assessment requires submission_id")
	}
	repo, err := contractRepository(u)
	if err != nil {
		return err
	}
	p, err := repo.GetSubmission(ctx, owner.Scope, *id)
	if err != nil {
		return err
	}
	if p.Material.WorkItemID != w.ID || w.CurrentContractID == nil || p.Material.ContractID != *w.CurrentContractID {
		return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "submission does not own this work")
	}
	c, err := repo.Get(ctx, owner.Scope, p.Material.ContractID)
	if err != nil {
		return err
	}
	if c.LatestSubmissionID == nil || *c.LatestSubmissionID != p.ID {
		return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "submission superseded")
	}
	for _, eid := range a.EvidenceIDs {
		found := false
		for _, ref := range p.Material.CriterionEvidence {
			if ref.CriterionID == a.CriterionID && ref.CriterionRevision == a.CriterionRevision {
				for _, pid := range ref.EvidenceIDs {
					if pid == eid {
						found = true
					}
				}
			}
		}
		if !found {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "assessment evidence is not bound to submitted criterion")
		}
	}
	a.SubmissionID = cloneIDPtr(id)
	a.SubmissionDigest = p.Digest
	return nil
}
