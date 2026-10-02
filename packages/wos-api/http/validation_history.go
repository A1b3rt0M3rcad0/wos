package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (h *Handler) getOutcomeCriterionHistory(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.getCriterionHistory(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindOutcome,
		ID:    scope.OutcomeID,
	})
}

func (h *Handler) getObjectiveCriterionHistory(w http.ResponseWriter, r *http.Request) {
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
	h.getCriterionHistory(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindObjective,
		ID:    id,
	})
}

func (h *Handler) getWorkItemCriterionHistory(w http.ResponseWriter, r *http.Request) {
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
	h.getCriterionHistory(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindWorkItem,
		ID:    id,
	})
}

func (h *Handler) getCriterionHistory(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	criterionID, err := parsePathID(r, "criterion_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetCriterionHistory(r.Context(), owner, criterionID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listOutcomeConclusions(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.listConclusions(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindOutcome,
		ID:    scope.OutcomeID,
	})
}

func (h *Handler) listObjectiveConclusions(w http.ResponseWriter, r *http.Request) {
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
	h.listConclusions(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindObjective,
		ID:    id,
	})
}

func (h *Handler) listWorkItemConclusions(w http.ResponseWriter, r *http.Request) {
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
	h.listConclusions(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindWorkItem,
		ID:    id,
	})
}

func (h *Handler) listConclusions(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	result, err := h.service.ListConclusions(r.Context(), owner)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getOutcomeConclusion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.getConclusion(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindOutcome,
		ID:    scope.OutcomeID,
	})
}

func (h *Handler) getObjectiveConclusion(w http.ResponseWriter, r *http.Request) {
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
	h.getConclusion(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindObjective,
		ID:    id,
	})
}

func (h *Handler) getWorkItemConclusion(w http.ResponseWriter, r *http.Request) {
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
	h.getConclusion(w, r, domain.EntityRef{
		Scope: scope,
		Kind:  domain.EntityKindWorkItem,
		ID:    id,
	})
}

func (h *Handler) getConclusion(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	conclusionID, err := parsePathID(r, "conclusion_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetConclusion(r.Context(), owner, conclusionID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
