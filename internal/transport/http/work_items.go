package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) createWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request createWorkItemRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	objectiveID, err := optionalPathID(request.ObjectiveID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CreateWorkItem(r.Context(), cc, application.CreateWorkItemCommand{
		Scope:       scope,
		Title:       request.Title,
		Description: request.Description,
		Priority:    request.Priority,
		Lifecycle:   request.Lifecycle,
		ObjectiveID: objectiveID,
		NotBefore:   request.NotBefore,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	w.Header().Set("Location", h.prefix+"/namespaces/"+scope.NamespaceID.String()+"/outcomes/"+scope.OutcomeID.String()+"/work-items/"+result.Value.ID.String())
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) updateWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request updateWorkItemRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.UpdateWorkItem(r.Context(), cc, application.UpdateWorkItemCommand{
		Scope:           scope,
		WorkItemID:      id,
		ExpectedVersion: expected,
		Title:           request.Title,
		Description:     request.Description,
		Priority:        request.Priority,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetWorkItem(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listWorkItems(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, collectionResponse[domain.WorkItem]{
		Items:           state.WorkItems,
		OutcomeRevision: state.OutcomeRevision,
	})
}

func (h *Handler) activateWorkItem(w http.ResponseWriter, r *http.Request) {
	h.workVersionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.WorkItem], error) {
		return h.service.ActivateWorkItem(r.Context(), cc, application.ActivateWorkItemCommand{Scope: scope, WorkItemID: id, ExpectedVersion: expected})
	})
}

func (h *Handler) deferWorkItem(w http.ResponseWriter, r *http.Request) {
	h.workVersionAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, cc domain.CommandContext) (application.MutationResult[domain.WorkItem], error) {
		return h.service.DeferWorkItem(r.Context(), cc, application.DeferWorkItemCommand{Scope: scope, WorkItemID: id, ExpectedVersion: expected})
	})
}

func (h *Handler) claimWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, id, request, cc, ok := h.workClaimInput(w, r)
	if !ok {
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ClaimWorkItem(r.Context(), cc, application.ClaimWorkItemCommand{
		Scope:           scope,
		WorkItemID:      id,
		ExpectedVersion: expected,
		TTL:             request.leaseTTL(),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) releaseWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request releaseRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	claimID, err := domain.ParseID(request.ClaimID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ReleaseWorkItem(r.Context(), cc, application.ReleaseWorkItemCommand{
		Scope:           scope,
		WorkItemID:      id,
		ExpectedVersion: expected,
		ClaimID:         claimID,
		FencingToken:    request.FencingToken,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) completeWorkItem(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request completeRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	claimID, err := domain.ParseID(request.ClaimID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.CompleteWorkItem(r.Context(), cc, application.CompleteWorkItemCommand{
		Scope:           scope,
		WorkItemID:      id,
		ExpectedVersion: expected,
		ClaimID:         claimID,
		FencingToken:    request.FencingToken,
		ResultSummary:   request.ResultSummary,
		Reason:          request.Reason,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) cancelWorkItem(w http.ResponseWriter, r *http.Request) {
	h.workReasonAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.WorkItem], error) {
		return h.service.CancelWorkItem(r.Context(), cc, application.CancelWorkItemCommand{Scope: scope, WorkItemID: id, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) reopenWorkItem(w http.ResponseWriter, r *http.Request) {
	h.workReasonAction(w, r, func(scope domain.Scope, id domain.ID, expected domain.Version, reason string, cc domain.CommandContext) (application.MutationResult[domain.WorkItem], error) {
		return h.service.ReopenWorkItem(r.Context(), cc, application.ReopenWorkItemCommand{Scope: scope, WorkItemID: id, ExpectedVersion: expected, Reason: reason})
	})
}

func (h *Handler) workVersionAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, domain.CommandContext) (application.MutationResult[domain.WorkItem], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
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
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) workReasonAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(domain.Scope, domain.ID, domain.Version, string, domain.CommandContext) (application.MutationResult[domain.WorkItem], error),
) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request reasonRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, domain.EntityKindWorkItem, id)
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
	setETag(w, domain.EntityKindWorkItem, result.Value.ID, result.Value.Version)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) workClaimInput(
	w http.ResponseWriter,
	r *http.Request,
) (domain.Scope, domain.ID, claimRequest, domain.CommandContext, bool) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", claimRequest{}, domain.CommandContext{}, false
	}
	id, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", claimRequest{}, domain.CommandContext{}, false
	}
	var request claimRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", claimRequest{}, domain.CommandContext{}, false
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return domain.Scope{}, "", claimRequest{}, domain.CommandContext{}, false
	}
	return scope, id, request, cc, true
}
