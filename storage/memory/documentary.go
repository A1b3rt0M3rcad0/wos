package memory

import (
	"context"
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type artifactRepository struct{ tx *transaction }

func (r artifactRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return domain.Artifact{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Artifact{}, err
	}
	value, ok := r.tx.artifacts[entityKey(scope, id)]
	if !ok {
		return domain.Artifact{}, domain.NewError(domain.ErrorCodeNotFound, "artifact not found")
	}
	return cloneArtifact(value), nil
}

func (r artifactRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	result := make([]domain.Artifact, 0)
	for _, value := range r.tx.artifacts {
		if value.Scope == scope {
			result = append(result, cloneArtifact(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result, nil
}

func (r artifactRepository) Insert(ctx context.Context, value domain.Artifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, ok := r.tx.artifacts[key]; ok {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "artifact already exists")
	}
	r.tx.artifacts[key] = cloneArtifact(value)
	return nil
}

func (r artifactRepository) Save(ctx context.Context, value domain.Artifact, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.artifacts[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "artifact not found")
	}
	if current.Version != expected || value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "artifact expected_version does not match")
	}
	r.tx.artifacts[key] = cloneArtifact(value)
	return nil
}

type evidenceRepository struct{ tx *transaction }

func (r evidenceRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return domain.Evidence{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Evidence{}, err
	}
	value, ok := r.tx.evidence[entityKey(scope, id)]
	if !ok {
		return domain.Evidence{}, domain.NewError(domain.ErrorCodeNotFound, "evidence not found")
	}
	return cloneEvidenceValue(value), nil
}

func (r evidenceRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	result := make([]domain.Evidence, 0)
	for _, value := range r.tx.evidence {
		if value.Scope == scope {
			result = append(result, cloneEvidenceValue(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result, nil
}

func (r evidenceRepository) Insert(ctx context.Context, value domain.Evidence) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, ok := r.tx.evidence[key]; ok {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "evidence already exists")
	}
	r.tx.evidence[key] = cloneEvidenceValue(value)
	return nil
}

func (r evidenceRepository) Save(ctx context.Context, value domain.Evidence, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.evidence[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "evidence not found")
	}
	if current.Version != expected || value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence expected_version does not match")
	}
	r.tx.evidence[key] = cloneEvidenceValue(value)
	return nil
}

type decisionRepository struct{ tx *transaction }

func (r decisionRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Decision, error) {
	if err := ctx.Err(); err != nil {
		return domain.Decision{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.Decision{}, err
	}
	value, ok := r.tx.decisions[entityKey(scope, id)]
	if !ok {
		return domain.Decision{}, domain.NewError(domain.ErrorCodeNotFound, "decision not found")
	}
	return cloneDecision(value), nil
}

func (r decisionRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Decision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	result := make([]domain.Decision, 0)
	for _, value := range r.tx.decisions {
		if value.Scope == scope {
			result = append(result, cloneDecision(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result, nil
}

func (r decisionRepository) Insert(ctx context.Context, value domain.Decision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, ok := r.tx.decisions[key]; ok {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "decision already exists")
	}
	r.tx.decisions[key] = cloneDecision(value)
	return nil
}

func (r decisionRepository) Save(ctx context.Context, value domain.Decision, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.decisions[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "decision not found")
	}
	if current.Version != expected || value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "decision expected_version does not match")
	}
	r.tx.decisions[key] = cloneDecision(value)
	return nil
}

type evidenceLinkRepository struct{ tx *transaction }

func (r evidenceLinkRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EvidenceLink, error) {
	if err := ctx.Err(); err != nil {
		return domain.EvidenceLink{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return domain.EvidenceLink{}, err
	}
	value, ok := r.tx.evidenceLinks[entityKey(scope, id)]
	if !ok {
		return domain.EvidenceLink{}, domain.NewError(domain.ErrorCodeNotFound, "evidence link not found")
	}
	return cloneEvidenceLink(value), nil
}

func (r evidenceLinkRepository) ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.EvidenceLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	result := make([]domain.EvidenceLink, 0)
	for _, value := range r.tx.evidenceLinks {
		if value.Scope == scope {
			result = append(result, cloneEvidenceLink(value))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result, nil
}

func (r evidenceLinkRepository) Insert(ctx context.Context, value domain.EvidenceLink) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	if _, ok := r.tx.evidenceLinks[key]; ok {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "evidence link already exists")
	}
	r.tx.evidenceLinks[key] = cloneEvidenceLink(value)
	return nil
}

func (r evidenceLinkRepository) Save(ctx context.Context, value domain.EvidenceLink, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	key := entityKey(value.Scope, value.ID)
	current, ok := r.tx.evidenceLinks[key]
	if !ok {
		return domain.NewError(domain.ErrorCodeNotFound, "evidence link not found")
	}
	if current.Version != expected || value.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "evidence link expected_version does not match")
	}
	r.tx.evidenceLinks[key] = cloneEvidenceLink(value)
	return nil
}
