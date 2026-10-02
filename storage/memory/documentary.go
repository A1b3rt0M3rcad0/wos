package memory

import (
	"context"
	"sort"
	"time"

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
	if !sameArtifactContent(current, value) {
		return domain.NewError(domain.ErrorCodeArtifact, "artifact content is immutable")
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
	if !sameEvidenceContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidence, "evidence observation is immutable")
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
	if current.Lifecycle != domain.DecisionLifecycleProposed && !sameDecisionContent(current, value) {
		return domain.NewError(domain.ErrorCodeDecision, "final decision content is immutable")
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
	if !sameEvidenceLinkContent(current, value) {
		return domain.NewError(domain.ErrorCodeEvidenceLink, "evidence link content is immutable")
	}
	r.tx.evidenceLinks[key] = cloneEvidenceLink(value)
	return nil
}

func sameArtifactContent(left, right domain.Artifact) bool {
	return left.ArtifactType == right.ArtifactType &&
		left.Name == right.Name &&
		left.URI == right.URI &&
		left.MediaType == right.MediaType &&
		left.Checksum == right.Checksum &&
		left.SourceVersion == right.SourceVersion &&
		left.ProducerRef == right.ProducerRef &&
		sameOptionalTime(left.ProducedAt, right.ProducedAt) &&
		left.RegisteredAt.Equal(right.RegisteredAt)
}

func sameEvidenceContent(left, right domain.Evidence) bool {
	return left.EvidenceType == right.EvidenceType &&
		left.Description == right.Description &&
		left.SourceRef == right.SourceRef &&
		left.ProducerRef == right.ProducerRef &&
		left.CapturedAt.Equal(right.CapturedAt) &&
		left.RegisteredAt.Equal(right.RegisteredAt) &&
		sameOptionalID(left.ArtifactID, right.ArtifactID) &&
		sameMeasurement(left.Measurement, right.Measurement) &&
		left.SourceVersion == right.SourceVersion &&
		left.Checksum == right.Checksum
}

func sameDecisionContent(left, right domain.Decision) bool {
	return left.Title == right.Title &&
		left.Proposal == right.Proposal &&
		left.ChosenAlternative == right.ChosenAlternative &&
		left.Rationale == right.Rationale &&
		sameStrings(left.Alternatives, right.Alternatives) &&
		sameOptionalActor(left.DecidedBy, right.DecidedBy) &&
		sameOptionalTime(left.DecidedAt, right.DecidedAt) &&
		sameOptionalID(left.SupersedesDecisionID, right.SupersedesDecisionID) &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func sameEvidenceLinkContent(left, right domain.EvidenceLink) bool {
	return left.EvidenceID == right.EvidenceID &&
		left.TargetRef == right.TargetRef &&
		sameOptionalID(left.CriterionID, right.CriterionID) &&
		left.Stance == right.Stance &&
		left.Rationale == right.Rationale &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func sameOptionalID(left, right *domain.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameOptionalActor(left, right *domain.ActorRef) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameMeasurement(left, right *domain.Measurement) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return string(left.Value) == string(right.Value) &&
		left.Unit == right.Unit &&
		left.Method == right.Method &&
		string(left.Conditions) == string(right.Conditions)
}
