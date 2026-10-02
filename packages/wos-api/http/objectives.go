package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (h *Handler) createObjective(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createObjectiveRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	parentID, err := optionalPathID(request.ParentObjectiveID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateObjective(r.Context(), cc, application.CreateObjectiveCommand{
		Scope:              scope,
		Title:              request.Title,
		Description:        request.Description,
		Priority:           request.Priority,
		RequiredForOutcome: request.RequiredForOutcome,
		ParentObjectiveID:  parentID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindObjective, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", h.prefix+"/namespaces/"+scope.NamespaceID.String()+"/outcomes/"+scope.OutcomeID.String()+"/objectives/"+result.Value.ID.String())
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) updateObjective(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "objective_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request updateObjectiveRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindObjective, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.UpdateObjective(r.Context(), cc, application.UpdateObjectiveCommand{
		Scope:              scope,
		ObjectiveID:        id,
		ExpectedVersion:    expected,
		Title:              request.Title,
		Description:        request.Description,
		Priority:           request.Priority,
		RequiredForOutcome: request.RequiredForOutcome,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindObjective, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getObjective(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "objective_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetObjective(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindObjective, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listObjectives(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	state, err := h.service.GetOutcomeState(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, collectionResponse[domain.Objective]{
		Items:           state.Objectives,
		OutcomeRevision: state.OutcomeRevision,
	})
}

func (h *Handler) startObjective(w http.ResponseWriter, r *http.Request) {
	h.objectiveVersionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.Objective], error) {
		return h.service.StartObjective(r.Context(), cc, application.StartObjectiveCommand{Scope: scope, ObjectiveID: id, ExpectedVersion: expected})
	})
}

func (h *Handler) achieveObjective(w http.ResponseWriter, r *http.Request) {
	h.objectiveReasonAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Objective], error) {
		return h.service.AchieveObjective(r.Context(), cc, application.AchieveObjectiveCommand{Scope: scope, ObjectiveID: id, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) cancelObjective(w http.ResponseWriter, r *http.Request) {
	h.objectiveReasonAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Objective], error) {
		return h.service.CancelObjective(r.Context(), cc, application.CancelObjectiveCommand{Scope: scope, ObjectiveID: id, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) reopenObjective(w http.ResponseWriter, r *http.Request) {
	h.objectiveReasonAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Objective], error) {
		return h.service.ReopenObjective(r.Context(), cc, application.ReopenObjectiveCommand{Scope: scope, ObjectiveID: id, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) objectiveVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, domain.CommandContext) (application.MutationResult[domain.Objective], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "objective_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindObjective, id)
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
	setETag(w, domain.EntityKindObjective, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) objectiveReasonAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, string, domain.CommandContext) (application.MutationResult[domain.Objective], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "objective_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindObjective, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, id, expected, request.Reason, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindObjective, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}
