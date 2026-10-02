package httptransport

import (
	"net/http"
	"strconv"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (h *Handler) createRoadmap(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createRoadmapRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateRoadmap(r.Context(), cc, application.CreateRoadmapCommand{
		Scope: scope, PlanScope: request.PlanScope, Title: request.Title,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindRoadmap, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", roadmapURL(h.prefix, scope, result.Value.ID))
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) listRoadmaps(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListRoadmaps(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Roadmap]{
		Items: values, OutcomeRevision: revision,
	})
}

func (h *Handler) getRoadmap(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetRoadmap(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindRoadmap, id, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) openRoadmapDraft(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request openRoadmapDraftRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.OpenRoadmapDraft(r.Context(), cc, application.OpenRoadmapDraftCommand{
		Scope: scope, RoadmapID: id, ExpectedVersion: expected,
		BaseRevisionNumber: request.BaseRevisionNumber,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeRoadmapMutation(w, result)
}

func (h *Handler) replaceRoadmapDraft(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request replaceRoadmapDraftRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ReplaceRoadmapDraft(r.Context(), cc, application.ReplaceRoadmapDraftCommand{
		Scope: scope, RoadmapID: id, ExpectedVersion: expected,
		ExpectedDraftVersion: request.ExpectedDraftVersion,
		Nodes:                request.Nodes, AfterLinks: request.AfterLinks,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeRoadmapMutation(w, result)
}

func (h *Handler) discardRoadmapDraft(w http.ResponseWriter, r *http.Request) {
	h.roadmapDraftVersionAction(w, r, func(
		scope domain.Scope, id domain.ID, expected domain.Version, draftVersion uint64, cc domain.CommandContext,
	) (application.MutationResult[domain.Roadmap], error) {
		return h.service.DiscardRoadmapDraft(r.Context(), cc, application.DiscardRoadmapDraftCommand{
			Scope: scope, RoadmapID: id, ExpectedVersion: expected, ExpectedDraftVersion: draftVersion,
		})
	})
}

func (h *Handler) publishRoadmapDraft(w http.ResponseWriter, r *http.Request) {
	h.roadmapDraftVersionAction(w, r, func(
		scope domain.Scope, id domain.ID, expected domain.Version, draftVersion uint64, cc domain.CommandContext,
	) (application.MutationResult[domain.Roadmap], error) {
		return h.service.PublishRoadmapDraft(r.Context(), cc, application.PublishRoadmapDraftCommand{
			Scope: scope, RoadmapID: id, ExpectedVersion: expected, ExpectedDraftVersion: draftVersion,
		})
	})
}

func (h *Handler) getRoadmapRevision(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	number, err := parseRoadmapRevisionNumber(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetRoadmapRevision(r.Context(), scope, id, number)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "public, immutable")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) activateRoadmapRevision(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	number, err := parseRoadmapRevisionNumber(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ActivateRoadmapRevision(r.Context(), cc, application.ActivateRoadmapRevisionCommand{
		Scope: scope, RoadmapID: id, ExpectedVersion: expected, RevisionNumber: number,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) deactivateRoadmapRevision(w http.ResponseWriter, r *http.Request) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	number, err := parseRoadmapRevisionNumber(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.DeactivateRoadmapRevision(r.Context(), cc, application.DeactivateRoadmapRevisionCommand{
		Scope: scope, RoadmapID: id, ExpectedVersion: expected, RevisionNumber: number,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) archiveRoadmap(w http.ResponseWriter, r *http.Request) {
	h.roadmapVersionAction(w, r, func(
		scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext,
	) (application.MutationResult[domain.Roadmap], error) {
		return h.service.ArchiveRoadmap(r.Context(), cc, application.ArchiveRoadmapCommand{
			Scope: scope, RoadmapID: id, ExpectedVersion: expected,
		})
	})
}

func (h *Handler) reopenRoadmap(w http.ResponseWriter, r *http.Request) {
	h.roadmapVersionAction(w, r, func(
		scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext,
	) (application.MutationResult[domain.Roadmap], error) {
		return h.service.ReopenRoadmap(r.Context(), cc, application.ReopenRoadmapCommand{
			Scope: scope, RoadmapID: id, ExpectedVersion: expected,
		})
	})
}

func (h *Handler) getActiveRoadmapSlot(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	planScope, err := parseRoadmapPlanScope(r, scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	slot, revision, err := h.service.GetActiveRoadmapSlot(r.Context(), scope, planScope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if slot == nil {
		writeError(w, r, domain.NewError(domain.ErrorCodeNotFound, "active roadmap slot not found"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, roadmapSlotReadResponse{Value: slot, OutcomeRevision: revision})
}

func (h *Handler) listRoadmapActivationHistory(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	planScope, err := parseRoadmapPlanScope(r, scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values, revision, err := h.service.ListRoadmapActivationHistory(r.Context(), scope, planScope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, collectionResponse[domain.RoadmapActivationRecord]{
		Items: values, OutcomeRevision: revision,
	})
}

func (h *Handler) roadmapDraftVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, uint64, domain.CommandContext) (application.MutationResult[domain.Roadmap], error),
) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request roadmapDraftVersionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	if request.ExpectedDraftVersion == 0 {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "expected_draft_version must be at least 1"))
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, request.ExpectedDraftVersion, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeRoadmapMutation(w, result)
}

func (h *Handler) roadmapVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, domain.CommandContext) (application.MutationResult[domain.Roadmap], error),
) {
	scope, id, err := parseRoadmapTarget(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindRoadmap, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeRoadmapMutation(w, result)
}

func writeRoadmapMutation(w http.ResponseWriter, result application.MutationResult[domain.Roadmap]) {
	setETag(w, domain.EntityKindRoadmap, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func parseRoadmapTarget(r *http.Request) (domain.Scope, domain.ID, error) {
	scope, err := parseScope(r)
	if err != nil {
		return domain.Scope{}, "", err
	}
	id, err := parsePathID(r, "roadmap_id")
	return scope, id, err
}

func parseRoadmapRevisionNumber(r *http.Request) (uint64, error) {
	value, err := strconv.ParseUint(r.PathValue("revision_number"), 10, 64)
	if err != nil || value == 0 {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "roadmap revision number must be a positive integer")
	}
	return value, nil
}

func parseRoadmapPlanScope(r *http.Request, scope domain.Scope) (domain.RoadmapPlanScope, error) {
	kind := domain.RoadmapScopeKind(r.PathValue("scope_kind"))
	id, err := domain.ParseID(r.PathValue("scope_id"))
	if err != nil {
		return domain.RoadmapPlanScope{}, err
	}
	value := domain.RoadmapPlanScope{Kind: kind, ID: id}
	if err := value.Validate(scope); err != nil {
		return domain.RoadmapPlanScope{}, err
	}
	return value, nil
}

func roadmapURL(prefix string, scope domain.Scope, id domain.ID) string {
	return prefix + "/namespaces/" + scope.NamespaceID.String() +
		"/outcomes/" + scope.OutcomeID.String() + "/roadmaps/" + id.String()
}
