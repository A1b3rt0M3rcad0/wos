package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (h *Handler) createRelation(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createRelationRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	if request.RelationType != domain.RelationTypeDependsOn {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidRelation, "Wave 06 relation creation supports depends_on only"))
		return
	}
	if request.Satisfaction != "" && request.Satisfaction != domain.DependencySatisfactionTargetCompleted {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidRelation, "depends_on satisfaction must be target_completed"))
		return
	}
	source, err := relationEndpoint(scope, request.SourceRef)
	if err != nil {
		writeError(w, r, err)
		return
	}
	target, err := relationEndpoint(scope, request.TargetRef)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.AddDependency(r.Context(), cc, application.AddDependencyCommand{
		Scope:     scope,
		SourceRef: source,
		TargetRef: target,
		Strength:  request.Strength,
		Reason:    request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindRelation, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", h.prefix+"/namespaces/"+scope.NamespaceID.String()+"/outcomes/"+scope.OutcomeID.String()+"/relations/"+result.Value.ID.String())
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getRelation(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "relation_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetRelation(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindRelation, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listRelations(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, revision, err := h.service.ListRelations(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Relation]{
		Items:           items,
		OutcomeRevision: revision,
	})
}

func (h *Handler) removeRelation(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "relation_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRelation, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RemoveDependency(r.Context(), cc, application.RemoveDependencyCommand{
		Scope:           scope,
		RelationID:      id,
		ExpectedVersion: expected,
		Reason:          request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindRelation, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func relationEndpoint(scope domain.Scope, request relationEndpointRequest) (domain.EntityRef, error) {
	id, err := domain.ParseID(request.ID)
	if err != nil {
		return domain.EntityRef{}, err
	}
	ref := domain.EntityRef{Scope: scope, Kind: request.Kind, ID: id}
	if err := ref.Validate(); err != nil {
		return domain.EntityRef{}, err
	}
	return ref, nil
}
