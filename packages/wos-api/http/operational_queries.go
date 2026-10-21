package httptransport

import (
	"net/http"
	"strconv"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

const maxHTTPCompletionSampleLimit = 20

func (h *Handler) getObjectiveCompletion(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	objectiveID, err := parsePathID(r, "objective_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > maxHTTPCompletionSampleLimit {
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "completion report limit must be between 1 and 20"))
			return
		}
	}
	result, err := h.service.GetObjectiveCompletionReport(r.Context(), scope, objectiveID, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	setETag(w, domain.EntityKindObjective, objectiveID, result.Value.ObjectiveVersion)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getTaskVerification(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	workItemID, err := parsePathID(r, "work_item_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetTaskVerificationView(r.Context(), scope, workItemID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	setETag(w, domain.EntityKindWorkItem, workItemID, result.TaskVersion)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getTaskDelivery(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	deliveryID, err := parsePathID(r, "delivery_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.service.GetTaskDelivery(r.Context(), scope, deliveryID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}
