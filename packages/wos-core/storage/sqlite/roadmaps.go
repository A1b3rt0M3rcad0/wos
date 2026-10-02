package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type roadmapRepository struct{ uow *unitOfWork }
type roadmapActivationStore struct{ uow *unitOfWork }

func (r roadmapRepository) Get(
	ctx context.Context,
	scope domain.Scope,
	id domain.ID,
) (domain.Roadmap, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return domain.Roadmap{}, err
	}
	var (
		version                              int64
		scopeKind, rawScopeID, title, life   string
		createdAt, updatedAt                 int64
		archivedAt                           sql.NullInt64
	)
	err := r.uow.tx.QueryRowContext(ctx, `
SELECT version, plan_scope_kind, plan_scope_id, title, lifecycle,
       created_at, updated_at, archived_at
FROM roadmaps
WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), id.String(),
	).Scan(
		&version, &scopeKind, &rawScopeID, &title, &life,
		&createdAt, &updatedAt, &archivedAt,
	)
	if err != nil {
		return domain.Roadmap{}, mapSQLError("get roadmap", err)
	}
	scopeID, err := domain.ParseID(rawScopeID)
	if err != nil {
		return domain.Roadmap{}, err
	}
	value := domain.Roadmap{
		ID:        id,
		Scope:     scope,
		Version:   domain.Version(version),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeKind(scopeKind), ID: scopeID},
		Title:     title,
		Lifecycle: domain.RoadmapLifecycle(life),
		CreatedAt: decodeTime(createdAt),
		UpdatedAt: decodeTime(updatedAt),
	}
	if archivedAt.Valid {
		at := decodeTime(archivedAt.Int64)
		value.ArchivedAt = &at
	}
	draft, err := loadRoadmapDraft(ctx, r.uow.tx, scope, id)
	if err != nil {
		return domain.Roadmap{}, err
	}
	value.Draft = draft
	revisions, err := loadRoadmapRevisions(ctx, r.uow.tx, scope, id)
	if err != nil {
		return domain.Roadmap{}, err
	}
	value.Revisions = revisions
	if err := value.Validate(); err != nil {
		return domain.Roadmap{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "persisted roadmap is invalid", err)
	}
	return value, nil
}

func (r roadmapRepository) ListByOutcome(
	ctx context.Context,
	scope domain.Scope,
) ([]domain.Roadmap, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := r.uow.tx.QueryContext(ctx, `
SELECT id
FROM roadmaps
WHERE namespace_id = ? AND outcome_id = ?
ORDER BY created_at, id`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
	)
	if err != nil {
		return nil, mapSQLError("list roadmaps", err)
	}
	defer rows.Close()
	ids := make([]domain.ID, 0)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.Roadmap, 0, len(ids))
	for _, id := range ids {
		value, err := r.Get(ctx, scope, id)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (r roadmapRepository) Insert(ctx context.Context, roadmap domain.Roadmap) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := roadmap.Validate(); err != nil {
		return err
	}
	if err := insertEntityRef(ctx, r.uow.tx, roadmap.Ref()); err != nil {
		return err
	}
	_, err := r.uow.tx.ExecContext(ctx, `
INSERT INTO roadmaps (
    id, namespace_id, outcome_id, kind, version, plan_scope_kind, plan_scope_id,
    title, lifecycle, created_at, updated_at, archived_at
) VALUES (?, ?, ?, 'roadmap', ?, ?, ?, ?, ?, ?, ?, ?)`,
		roadmap.ID.String(),
		roadmap.Scope.NamespaceID.String(),
		roadmap.Scope.OutcomeID.String(),
		int64(roadmap.Version),
		string(roadmap.PlanScope.Kind),
		roadmap.PlanScope.ID.String(),
		roadmap.Title,
		string(roadmap.Lifecycle),
		encodeTime(roadmap.CreatedAt),
		encodeTime(roadmap.UpdatedAt),
		encodeOptionalTime(roadmap.ArchivedAt),
	)
	if err != nil {
		return mapSQLError("insert roadmap", err)
	}
	return syncRoadmapNested(ctx, r.uow.tx, roadmap)
}

func (r roadmapRepository) Save(
	ctx context.Context,
	roadmap domain.Roadmap,
	expected domain.Version,
) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := roadmap.Validate(); err != nil {
		return err
	}
	if roadmap.Version != expected+1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "roadmap version must advance exactly once per save")
	}

	current, err := r.Get(ctx, roadmap.Scope, roadmap.ID)
	if err != nil {
		return err
	}
	if current.Version != expected {
		return domain.NewError(domain.ErrorCodeVersionConflict, "roadmap expected_version does not match")
	}
	if len(roadmap.Revisions) < len(current.Revisions) {
		return domain.NewError(domain.ErrorCodeRoadmap, "published roadmap revision history is append-only")
	}
	for i := range current.Revisions {
		if !reflect.DeepEqual(current.Revisions[i], roadmap.Revisions[i]) {
			return domain.NewError(domain.ErrorCodeRoadmap, "published roadmap revision is immutable")
		}
	}

	result, err := r.uow.tx.ExecContext(ctx, `
UPDATE roadmaps
SET version = ?, title = ?, lifecycle = ?, updated_at = ?, archived_at = ?
WHERE namespace_id = ? AND outcome_id = ? AND id = ? AND version = ?`,
		int64(roadmap.Version),
		roadmap.Title,
		string(roadmap.Lifecycle),
		encodeTime(roadmap.UpdatedAt),
		encodeOptionalTime(roadmap.ArchivedAt),
		roadmap.Scope.NamespaceID.String(),
		roadmap.Scope.OutcomeID.String(),
		roadmap.ID.String(),
		int64(expected),
	)
	if err != nil {
		return mapSQLError("save roadmap", err)
	}
	if err := requireVersionedUpdate(
		ctx, r.uow.tx, result, "roadmaps",
		roadmap.Scope.NamespaceID, roadmap.Scope.OutcomeID, roadmap.ID, expected,
	); err != nil {
		return err
	}
	return syncRoadmapNested(ctx, r.uow.tx, roadmap)
}

func syncRoadmapNested(ctx context.Context, tx *sql.Tx, roadmap domain.Roadmap) error {
	if roadmap.Draft == nil {
		if _, err := tx.ExecContext(ctx, `
DELETE FROM roadmap_drafts
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ?`,
			roadmap.Scope.NamespaceID.String(),
			roadmap.Scope.OutcomeID.String(),
			roadmap.ID.String(),
		); err != nil {
			return mapSQLError("delete roadmap draft", err)
		}
	} else {
		nodesJSON, err := marshalJSON(roadmap.Draft.Nodes)
		if err != nil {
			return err
		}
		linksJSON, err := marshalJSON(roadmap.Draft.AfterLinks)
		if err != nil {
			return err
		}
		var base any
		if roadmap.Draft.BaseRevisionNumber != nil {
			base = int64(*roadmap.Draft.BaseRevisionNumber)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO roadmap_drafts (
    namespace_id, outcome_id, roadmap_id, draft_version, base_revision_number,
    lifecycle, nodes_json, after_links_json, created_at, updated_at, discarded_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(namespace_id, outcome_id, roadmap_id)
DO UPDATE SET
    draft_version = excluded.draft_version,
    base_revision_number = excluded.base_revision_number,
    lifecycle = excluded.lifecycle,
    nodes_json = excluded.nodes_json,
    after_links_json = excluded.after_links_json,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    discarded_at = excluded.discarded_at`,
			roadmap.Scope.NamespaceID.String(),
			roadmap.Scope.OutcomeID.String(),
			roadmap.ID.String(),
			int64(roadmap.Draft.DraftVersion),
			base,
			string(roadmap.Draft.Lifecycle),
			nodesJSON,
			linksJSON,
			encodeTime(roadmap.Draft.CreatedAt),
			encodeTime(roadmap.Draft.UpdatedAt),
			encodeOptionalTime(roadmap.Draft.DiscardedAt),
		); err != nil {
			return mapSQLError("sync roadmap draft", err)
		}
	}

	for _, revision := range roadmap.Revisions {
		if err := insertRoadmapRevision(ctx, tx, roadmap.Scope, roadmap.ID, revision); err != nil {
			return err
		}
	}
	return nil
}

func insertRoadmapRevision(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	roadmapID domain.ID,
	revision domain.RoadmapRevision,
) error {
	actorJSON, err := marshalJSON(revision.PublishedBy)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO roadmap_revisions (
    namespace_id, outcome_id, roadmap_id, revision_number,
    content_hash, published_by_json, published_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
		int64(revision.RevisionNumber),
		revision.ContentHash,
		actorJSON,
		encodeTime(revision.PublishedAt),
	)
	if err != nil {
		return mapSQLError("insert roadmap revision", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		persisted, err := loadOneRoadmapRevision(ctx, tx, scope, roadmapID, revision.RevisionNumber)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(persisted, revision) {
			return domain.NewError(domain.ErrorCodeRoadmap, "published roadmap revision differs from immutable persisted history")
		}
		return nil
	}

	for _, node := range revision.Nodes {
		targetJSON, err := nullableJSON(node.TargetRef)
		if err != nil {
			return err
		}
		criterionRefsJSON, err := marshalJSON(node.CriterionRefs)
		if err != nil {
			return err
		}
		criterionSnapshotsJSON, err := marshalJSON(node.CriterionSnapshots)
		if err != nil {
			return err
		}
		referenceSnapshotJSON, err := nullableJSON(node.ReferenceSnapshot)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO roadmap_revision_nodes (
    namespace_id, outcome_id, roadmap_id, revision_number,
    node_key, node_type, parent_node_key, target_ref_json, title, position,
    criterion_refs_json, criterion_snapshots_json,
    planned_start, planned_end, reference_snapshot_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			scope.NamespaceID.String(),
			scope.OutcomeID.String(),
			roadmapID.String(),
			int64(revision.RevisionNumber),
			node.NodeKey,
			string(node.NodeType),
			nullableString(node.ParentNodeKey),
			targetJSON,
			node.Title,
			node.Position,
			criterionRefsJSON,
			criterionSnapshotsJSON,
			encodeOptionalTime(node.PlannedStart),
			encodeOptionalTime(node.PlannedEnd),
			referenceSnapshotJSON,
		); err != nil {
			return mapSQLError("insert roadmap revision node", err)
		}
	}
	for _, link := range revision.AfterLinks {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO roadmap_revision_after_links (
    namespace_id, outcome_id, roadmap_id, revision_number,
    node_key, after_node_key
) VALUES (?, ?, ?, ?, ?, ?)`,
			scope.NamespaceID.String(),
			scope.OutcomeID.String(),
			roadmapID.String(),
			int64(revision.RevisionNumber),
			link.NodeKey,
			link.AfterNodeKey,
		); err != nil {
			return mapSQLError("insert roadmap after link", err)
		}
	}
	for _, dep := range revision.DependencySnapshots {
		dependentJSON, err := marshalJSON(dep.DependentRef)
		if err != nil {
			return err
		}
		prerequisiteJSON, err := marshalJSON(dep.PrerequisiteRef)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO roadmap_revision_dependencies (
    namespace_id, outcome_id, roadmap_id, revision_number,
    dependent_ref_json, prerequisite_ref_json, strength, satisfaction
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			scope.NamespaceID.String(),
			scope.OutcomeID.String(),
			roadmapID.String(),
			int64(revision.RevisionNumber),
			dependentJSON,
			prerequisiteJSON,
			string(dep.Strength),
			string(dep.Satisfaction),
		); err != nil {
			return mapSQLError("insert roadmap dependency snapshot", err)
		}
	}
	return nil
}

func loadRoadmapDraft(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	roadmapID domain.ID,
) (*domain.RoadmapDraft, error) {
	var (
		draftVersion                      int64
		baseRevision                      sql.NullInt64
		lifecycle, nodesJSON, linksJSON   string
		createdAt, updatedAt               int64
		discardedAt                        sql.NullInt64
	)
	err := tx.QueryRowContext(ctx, `
SELECT draft_version, base_revision_number, lifecycle, nodes_json, after_links_json,
       created_at, updated_at, discarded_at
FROM roadmap_drafts
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ?`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
	).Scan(
		&draftVersion, &baseRevision, &lifecycle, &nodesJSON, &linksJSON,
		&createdAt, &updatedAt, &discardedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapSQLError("load roadmap draft", err)
	}
	value := &domain.RoadmapDraft{
		DraftVersion: uint64(draftVersion),
		Lifecycle:    domain.RoadmapDraftLifecycle(lifecycle),
		CreatedAt:    decodeTime(createdAt),
		UpdatedAt:    decodeTime(updatedAt),
	}
	if baseRevision.Valid {
		number := uint64(baseRevision.Int64)
		value.BaseRevisionNumber = &number
	}
	if discardedAt.Valid {
		at := decodeTime(discardedAt.Int64)
		value.DiscardedAt = &at
	}
	if err := unmarshalJSON(nodesJSON, &value.Nodes); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(linksJSON, &value.AfterLinks); err != nil {
		return nil, err
	}
	return value, nil
}

func loadRoadmapRevisions(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	roadmapID domain.ID,
) ([]domain.RoadmapRevision, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT revision_number
FROM roadmap_revisions
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ?
ORDER BY revision_number`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
	)
	if err != nil {
		return nil, mapSQLError("list roadmap revisions", err)
	}
	defer rows.Close()
	numbers := make([]uint64, 0)
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		numbers = append(numbers, uint64(value))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]domain.RoadmapRevision, 0, len(numbers))
	for _, number := range numbers {
		value, err := loadOneRoadmapRevision(ctx, tx, scope, roadmapID, number)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func loadOneRoadmapRevision(
	ctx context.Context,
	tx *sql.Tx,
	scope domain.Scope,
	roadmapID domain.ID,
	revisionNumber uint64,
) (domain.RoadmapRevision, error) {
	var contentHash, actorJSON string
	var publishedAt int64
	err := tx.QueryRowContext(ctx, `
SELECT content_hash, published_by_json, published_at
FROM roadmap_revisions
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ? AND revision_number = ?`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
		int64(revisionNumber),
	).Scan(&contentHash, &actorJSON, &publishedAt)
	if err != nil {
		return domain.RoadmapRevision{}, mapSQLError("get roadmap revision", err)
	}
	var actor domain.ActorRef
	if err := unmarshalJSON(actorJSON, &actor); err != nil {
		return domain.RoadmapRevision{}, err
	}
	value := domain.RoadmapRevision{
		RevisionNumber: revisionNumber,
		ContentHash:    contentHash,
		PublishedBy:    actor,
		PublishedAt:    decodeTime(publishedAt),
	}

	nodeRows, err := tx.QueryContext(ctx, `
SELECT node_key, node_type, parent_node_key, target_ref_json, title, position,
       criterion_refs_json, criterion_snapshots_json,
       planned_start, planned_end, reference_snapshot_json
FROM roadmap_revision_nodes
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ? AND revision_number = ?
ORDER BY node_key`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
		int64(revisionNumber),
	)
	if err != nil {
		return domain.RoadmapRevision{}, mapSQLError("load roadmap revision nodes", err)
	}
	for nodeRows.Next() {
		var (
			nodeKey, nodeType, title, criterionRefsJSON, criterionSnapshotsJSON string
			parent, targetJSON, referenceJSON                                  sql.NullString
			position                                                           int
			plannedStart, plannedEnd                                            sql.NullInt64
		)
		if err := nodeRows.Scan(
			&nodeKey, &nodeType, &parent, &targetJSON, &title, &position,
			&criterionRefsJSON, &criterionSnapshotsJSON,
			&plannedStart, &plannedEnd, &referenceJSON,
		); err != nil {
			nodeRows.Close()
			return domain.RoadmapRevision{}, err
		}
		node := domain.RoadmapNode{
			NodeKey: nodeKey, NodeType: domain.RoadmapNodeType(nodeType),
			Title: title, Position: position,
		}
		if parent.Valid {
			node.ParentNodeKey = parent.String
		}
		if targetJSON.Valid {
			var ref domain.EntityRef
			if err := unmarshalJSON(targetJSON.String, &ref); err != nil {
				nodeRows.Close()
				return domain.RoadmapRevision{}, err
			}
			node.TargetRef = &ref
		}
		if err := unmarshalJSON(criterionRefsJSON, &node.CriterionRefs); err != nil {
			nodeRows.Close()
			return domain.RoadmapRevision{}, err
		}
		if err := unmarshalJSON(criterionSnapshotsJSON, &node.CriterionSnapshots); err != nil {
			nodeRows.Close()
			return domain.RoadmapRevision{}, err
		}
		if plannedStart.Valid {
			at := decodeTime(plannedStart.Int64)
			node.PlannedStart = &at
		}
		if plannedEnd.Valid {
			at := decodeTime(plannedEnd.Int64)
			node.PlannedEnd = &at
		}
		if referenceJSON.Valid {
			var snapshot domain.RoadmapReferenceSnapshot
			if err := unmarshalJSON(referenceJSON.String, &snapshot); err != nil {
				nodeRows.Close()
				return domain.RoadmapRevision{}, err
			}
			node.ReferenceSnapshot = &snapshot
		}
		value.Nodes = append(value.Nodes, node)
	}
	if err := nodeRows.Close(); err != nil {
		return domain.RoadmapRevision{}, err
	}

	linkRows, err := tx.QueryContext(ctx, `
SELECT node_key, after_node_key
FROM roadmap_revision_after_links
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ? AND revision_number = ?
ORDER BY node_key, after_node_key`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
		int64(revisionNumber),
	)
	if err != nil {
		return domain.RoadmapRevision{}, mapSQLError("load roadmap after links", err)
	}
	for linkRows.Next() {
		var item domain.RoadmapAfterLink
		if err := linkRows.Scan(&item.NodeKey, &item.AfterNodeKey); err != nil {
			linkRows.Close()
			return domain.RoadmapRevision{}, err
		}
		value.AfterLinks = append(value.AfterLinks, item)
	}
	if err := linkRows.Close(); err != nil {
		return domain.RoadmapRevision{}, err
	}

	depRows, err := tx.QueryContext(ctx, `
SELECT dependent_ref_json, prerequisite_ref_json, strength, satisfaction
FROM roadmap_revision_dependencies
WHERE namespace_id = ? AND outcome_id = ? AND roadmap_id = ? AND revision_number = ?
ORDER BY dependent_ref_json, prerequisite_ref_json`,
		scope.NamespaceID.String(),
		scope.OutcomeID.String(),
		roadmapID.String(),
		int64(revisionNumber),
	)
	if err != nil {
		return domain.RoadmapRevision{}, mapSQLError("load roadmap dependency snapshots", err)
	}
	for depRows.Next() {
		var dependentJSON, prerequisiteJSON, strength, satisfaction string
		if err := depRows.Scan(&dependentJSON, &prerequisiteJSON, &strength, &satisfaction); err != nil {
			depRows.Close()
			return domain.RoadmapRevision{}, err
		}
		var dependent, prerequisite domain.EntityRef
		if err := unmarshalJSON(dependentJSON, &dependent); err != nil {
			depRows.Close()
			return domain.RoadmapRevision{}, err
		}
		if err := unmarshalJSON(prerequisiteJSON, &prerequisite); err != nil {
			depRows.Close()
			return domain.RoadmapRevision{}, err
		}
		value.DependencySnapshots = append(value.DependencySnapshots, domain.RoadmapDependencySnapshot{
			DependentRef: dependent, PrerequisiteRef: prerequisite,
			Strength: domain.DependencyStrength(strength),
			Satisfaction: domain.DependencySatisfaction(satisfaction),
		})
	}
	if err := depRows.Close(); err != nil {
		return domain.RoadmapRevision{}, err
	}
	if err := value.Validate(scope); err != nil {
		return domain.RoadmapRevision{}, err
	}
	return value, nil
}

func (s roadmapActivationStore) GetActive(
	ctx context.Context,
	scope domain.Scope,
	planScope domain.RoadmapPlanScope,
) (*domain.RoadmapActiveSlot, error) {
	if err := s.uow.ensureOpen(); err != nil {
		return nil, err
	}
	var rawRoadmapID, actorJSON string
	var revisionNumber, activatedAt int64
	err := s.uow.tx.QueryRowContext(ctx, `
SELECT roadmap_id, revision_number, activated_by_json, activated_at
FROM roadmap_active_slots
WHERE namespace_id = ? AND outcome_id = ? AND plan_scope_kind = ? AND plan_scope_id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
		string(planScope.Kind), planScope.ID.String(),
	).Scan(&rawRoadmapID, &revisionNumber, &actorJSON, &activatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapSQLError("get active roadmap slot", err)
	}
	roadmapID, err := domain.ParseID(rawRoadmapID)
	if err != nil {
		return nil, err
	}
	var actor domain.ActorRef
	if err := unmarshalJSON(actorJSON, &actor); err != nil {
		return nil, err
	}
	value := &domain.RoadmapActiveSlot{
		Scope: scope, PlanScope: planScope,
		RoadmapID: roadmapID, RevisionNumber: uint64(revisionNumber),
		ActivatedBy: actor, ActivatedAt: decodeTime(activatedAt),
	}
	return value, value.Validate()
}

func (s roadmapActivationStore) SetActive(
	ctx context.Context,
	slot domain.RoadmapActiveSlot,
	history []domain.RoadmapActivationRecord,
) error {
	if err := s.uow.ensureOpen(); err != nil {
		return err
	}
	if err := slot.Validate(); err != nil {
		return err
	}
	actorJSON, err := marshalJSON(slot.ActivatedBy)
	if err != nil {
		return err
	}
	if _, err := s.uow.tx.ExecContext(ctx, `
INSERT INTO roadmap_active_slots (
    namespace_id, outcome_id, plan_scope_kind, plan_scope_id,
    roadmap_id, revision_number, activated_by_json, activated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(namespace_id, outcome_id, plan_scope_kind, plan_scope_id)
DO UPDATE SET
    roadmap_id = excluded.roadmap_id,
    revision_number = excluded.revision_number,
    activated_by_json = excluded.activated_by_json,
    activated_at = excluded.activated_at`,
		slot.Scope.NamespaceID.String(),
		slot.Scope.OutcomeID.String(),
		string(slot.PlanScope.Kind),
		slot.PlanScope.ID.String(),
		slot.RoadmapID.String(),
		int64(slot.RevisionNumber),
		actorJSON,
		encodeTime(slot.ActivatedAt),
	); err != nil {
		return mapSQLError("set active roadmap slot", err)
	}
	for _, record := range history {
		if err := insertRoadmapActivationRecord(ctx, s.uow.tx, record); err != nil {
			return err
		}
	}
	return nil
}

func (s roadmapActivationStore) ClearActive(
	ctx context.Context,
	expected domain.RoadmapActiveSlot,
	record domain.RoadmapActivationRecord,
) error {
	if err := s.uow.ensureOpen(); err != nil {
		return err
	}
	result, err := s.uow.tx.ExecContext(ctx, `
DELETE FROM roadmap_active_slots
WHERE namespace_id = ? AND outcome_id = ? AND plan_scope_kind = ? AND plan_scope_id = ?
  AND roadmap_id = ? AND revision_number = ?`,
		expected.Scope.NamespaceID.String(),
		expected.Scope.OutcomeID.String(),
		string(expected.PlanScope.Kind),
		expected.PlanScope.ID.String(),
		expected.RoadmapID.String(),
		int64(expected.RevisionNumber),
	)
	if err != nil {
		return mapSQLError("clear active roadmap slot", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return domain.NewError(domain.ErrorCodeVersionConflict, "active roadmap slot changed")
	}
	return insertRoadmapActivationRecord(ctx, s.uow.tx, record)
}

func (s roadmapActivationStore) ListHistory(
	ctx context.Context,
	scope domain.Scope,
	planScope domain.RoadmapPlanScope,
) ([]domain.RoadmapActivationRecord, error) {
	if err := s.uow.ensureOpen(); err != nil {
		return nil, err
	}
	rows, err := s.uow.tx.QueryContext(ctx, `
SELECT id, action, roadmap_id, revision_number,
       replacement_roadmap_id, replacement_revision_number,
       actor_json, recorded_at
FROM roadmap_activation_history
WHERE namespace_id = ? AND outcome_id = ? AND plan_scope_kind = ? AND plan_scope_id = ?
ORDER BY recorded_at, id`,
		scope.NamespaceID.String(), scope.OutcomeID.String(),
		string(planScope.Kind), planScope.ID.String(),
	)
	if err != nil {
		return nil, mapSQLError("list roadmap activation history", err)
	}
	defer rows.Close()
	result := make([]domain.RoadmapActivationRecord, 0)
	for rows.Next() {
		var (
			rawID, action, rawRoadmapID, actorJSON string
			revisionNumber, recordedAt             int64
			replacementRoadmapID                   sql.NullString
			replacementRevision                    sql.NullInt64
		)
		if err := rows.Scan(
			&rawID, &action, &rawRoadmapID, &revisionNumber,
			&replacementRoadmapID, &replacementRevision,
			&actorJSON, &recordedAt,
		); err != nil {
			return nil, err
		}
		id, err := domain.ParseID(rawID)
		if err != nil {
			return nil, err
		}
		roadmapID, err := domain.ParseID(rawRoadmapID)
		if err != nil {
			return nil, err
		}
		var actor domain.ActorRef
		if err := unmarshalJSON(actorJSON, &actor); err != nil {
			return nil, err
		}
		record := domain.RoadmapActivationRecord{
			ID: id, Scope: scope, PlanScope: planScope,
			Action: domain.RoadmapActivationAction(action),
			RoadmapID: roadmapID, RevisionNumber: uint64(revisionNumber),
			Actor: actor, RecordedAt: decodeTime(recordedAt),
		}
		if replacementRoadmapID.Valid {
			replacementID, err := domain.ParseID(replacementRoadmapID.String)
			if err != nil {
				return nil, err
			}
			record.Replacement = &domain.RoadmapRevisionPointer{
				RoadmapID: replacementID,
				RevisionNumber: uint64(replacementRevision.Int64),
			}
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func insertRoadmapActivationRecord(
	ctx context.Context,
	tx *sql.Tx,
	record domain.RoadmapActivationRecord,
) error {
	if err := record.Validate(); err != nil {
		return err
	}
	actorJSON, err := marshalJSON(record.Actor)
	if err != nil {
		return err
	}
	var replacementID, replacementRevision any
	if record.Replacement != nil {
		replacementID = record.Replacement.RoadmapID.String()
		replacementRevision = int64(record.Replacement.RevisionNumber)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO roadmap_activation_history (
    id, namespace_id, outcome_id, plan_scope_kind, plan_scope_id,
    action, roadmap_id, revision_number,
    replacement_roadmap_id, replacement_revision_number,
    actor_json, recorded_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID.String(),
		record.Scope.NamespaceID.String(),
		record.Scope.OutcomeID.String(),
		string(record.PlanScope.Kind),
		record.PlanScope.ID.String(),
		string(record.Action),
		record.RoadmapID.String(),
		int64(record.RevisionNumber),
		replacementID,
		replacementRevision,
		actorJSON,
		encodeTime(record.RecordedAt),
	)
	return mapSQLError("insert roadmap activation history", err)
}

func nullableJSON(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch typed := value.(type) {
	case *domain.EntityRef:
		if typed == nil {
			return nil, nil
		}
	case *domain.RoadmapReferenceSnapshot:
		if typed == nil {
			return nil, nil
		}
	}
	return marshalJSON(value)
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func sortRoadmapRevisions(values []domain.RoadmapRevision) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].RevisionNumber < values[j].RevisionNumber
	})
}
