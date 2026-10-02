package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) createOutcome(w http.ResponseWriter, r *http.Request) {
	namespaceID, err := domain.ParseID(r.PathValue("namespace_id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createOutcomeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	commandContext, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateOutcome(r.Context(), commandContext, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        request.Title,
		Description:  request.Description,
		DesiredState: request.DesiredState,
		Priority:     request.Priority,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindOutcome, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", h.prefix+"/namespaces/"+namespaceID.String()+"/outcomes/"+result.Value.ID.String())
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) getOutcome(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetOutcome(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindOutcome, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) updateOutcome(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request updateOutcomeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindOutcome, scope.OutcomeID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.UpdateOutcome(r.Context(), cc, application.UpdateOutcomeCommand{
		Scope:           scope,
		ExpectedVersion: expected,
		Title:           request.Title,
		Description:     request.Description,
		DesiredState:    request.DesiredState,
		Priority:        request.Priority,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindOutcome, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getOutcomeState(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetOutcomeState(r.Context(), scope)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) activateOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeVersionAction(w, r, func(scope domain.Scope, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.ActivateOutcome(r.Context(), cc, application.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: expected})
	})
}

func (h *Handler) archiveOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeVersionAction(w, r, func(scope domain.Scope, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.ArchiveOutcome(r.Context(), cc, application.ArchiveOutcomeCommand{Scope: scope, ExpectedVersion: expected})
	})
}

func (h *Handler) achieveOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeReasonAction(w, r, func(scope domain.Scope, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.AchieveOutcome(r.Context(), cc, application.AchieveOutcomeCommand{Scope: scope, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) failOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeReasonAction(w, r, func(scope domain.Scope, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.FailOutcome(r.Context(), cc, application.FailOutcomeCommand{Scope: scope, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) abandonOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeReasonAction(w, r, func(scope domain.Scope, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.AbandonOutcome(r.Context(), cc, application.AbandonOutcomeCommand{Scope: scope, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) reopenOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeReasonAction(w, r, func(scope domain.Scope, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.ReopenOutcome(r.Context(), cc, application.ReopenOutcomeCommand{Scope: scope, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) unarchiveOutcome(w http.ResponseWriter, r *http.Request) {
	h.outcomeReasonAction(w, r, func(scope domain.Scope, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.Outcome], error) {
		return h.service.UnarchiveOutcome(r.Context(), cc, application.UnarchiveOutcomeCommand{Scope: scope, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) outcomeVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.Version, domain.CommandContext) (application.MutationResult[domain.Outcome], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindOutcome, scope.OutcomeID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, expected, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindOutcome, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) outcomeReasonAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.Version, string, domain.CommandContext) (application.MutationResult[domain.Outcome], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindOutcome, scope.OutcomeID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := action(scope, expected, request.Reason, cc)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindOutcome, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}
