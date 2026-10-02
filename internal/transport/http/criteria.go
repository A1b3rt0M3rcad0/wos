package httptransport

import (
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func (h *Handler) addOutcomeCriterion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.addCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID})
}

func (h *Handler) addObjectiveCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.addCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: id})
}

func (h *Handler) addWorkItemCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.addCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: id})
}

func (h *Handler) addCriterion(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	var request criterionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, owner.Kind, owner.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.AddCriterion(r.Context(), cc, application.AddCriterionCommand{
		Owner:            owner,
		ExpectedVersion:  expected,
		Title:            request.Title,
		Description:      request.Description,
		Required:         request.Required,
		VerificationMode: request.VerificationMode,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, owner.Kind, owner.ID, expected+1)
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) reviseOutcomeCriterion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.reviseCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID})
}

func (h *Handler) reviseObjectiveCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.reviseCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: id})
}

func (h *Handler) reviseWorkItemCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.reviseCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: id})
}

func (h *Handler) reviseCriterion(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	criterionID, err := parsePathID(r, "criterion_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request criterionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, owner.Kind, owner.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.ReviseCriterion(r.Context(), cc, application.ReviseCriterionCommand{
		Owner:            owner,
		CriterionID:      criterionID,
		ExpectedVersion:  expected,
		Title:            request.Title,
		Description:      request.Description,
		Required:         request.Required,
		VerificationMode: request.VerificationMode,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, owner.Kind, owner.ID, expected+1)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) retireOutcomeCriterion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.retireCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID})
}

func (h *Handler) retireObjectiveCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.retireCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: id})
}

func (h *Handler) retireWorkItemCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.retireCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: id})
}

func (h *Handler) retireCriterion(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	criterionID, err := parsePathID(r, "criterion_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request versionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, owner.Kind, owner.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.RetireCriterion(r.Context(), cc, application.RetireCriterionCommand{
		Owner:           owner,
		CriterionID:     criterionID,
		ExpectedVersion: expected,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, owner.Kind, owner.ID, expected+1)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) assessOutcomeCriterion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.assessCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID})
}

func (h *Handler) assessObjectiveCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.assessCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: id})
}

func (h *Handler) assessWorkItemCriterion(w http.ResponseWriter, r *http.Request) {
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
	h.assessCriterion(w, r, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: id})
}

func (h *Handler) assessCriterion(w http.ResponseWriter, r *http.Request, owner domain.EntityRef) {
	criterionID, err := parsePathID(r, "criterion_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var request assessmentRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, r, err)
		return
	}
	expected, err := expectedVersion(r, request.ExpectedVersion, owner.Kind, owner.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cc, err := h.commandContext(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.AttestCriterion(r.Context(), cc, application.AttestCriterionCommand{
		Owner:             owner,
		CriterionID:       criterionID,
		CriterionRevision: request.CriterionRevision,
		ExpectedVersion:   expected,
		Result:            request.Result,
		Rationale:         request.Rationale,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	setETag(w, owner.Kind, owner.ID, expected+1)
	writeJSON(w, http.StatusCreated, result)
}
