package httptransport

import (
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
)

func (h *Handler) deliveryHealth(w http.ResponseWriter, r *http.Request) {
	ns, e := parsePathID(r, "namespace_id")
	if e != nil {
		writeError(w, r, e)
		return
	}
	query := a.DeliveryHealthQuery{NamespaceID: ns}
	for key, values := range r.URL.Query() {
		if key != "outcome_id" || len(values) != 1 {
			writeError(w, r, d.NewError(d.ErrorCodeInvalidArgument, "delivery health accepts one optional outcome_id"))
			return
		}
	}
	if value := r.URL.Query().Get("outcome_id"); value != "" {
		id, e := d.ParseID(value)
		if e != nil {
			writeError(w, r, e)
			return
		}
		query.OutcomeID = &id
	}
	result, e := h.service.GetDeliveryHealth(r.Context(), query)
	if e != nil {
		writeError(w, r, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, result)
}
