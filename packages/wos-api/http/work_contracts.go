package httptransport

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/http"
)

func (h *Handler) executionCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, application.ContractCapabilities())
}
func (h *Handler) getWorkContract(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "contract_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.GetWorkContract(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.PathValue("section") == "spec" {
		writeJSON(w, 200, map[string]any{"contract_id": id, "spec_digest": v.Value.Contract.SpecDigest, "spec": v.Value.Contract.Spec, "snapshot_complete": true})
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) listWorkContracts(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	f := ports.ContractFilter{Limit: limit, HolderPrincipalID: q.Get("holder_principal_id"), Status: domain.ContractStatus(q.Get("status"))}
	if q.Get("work_item_id") != "" {
		f.WorkItemID, err = domain.ParseID(q.Get("work_item_id"))
		if err != nil {
			writeError(w, r, err)
			return
		}
	}
	v, err := h.service.WorkContractHistory(r.Context(), scope, f, q.Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, v)
}
func (h *Handler) contractRecords(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("section") == "spec" {
		h.getWorkContract(w, r)
		return
	}
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "contract_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.ContractRecords(r.Context(), scope, id, r.PathValue("section"), limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, v)
}
func (h *Handler) getWorkSubmission(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "submission_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.GetWorkSubmission(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, v)
}
func (h *Handler) getCommandReceipt(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "command_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.GetCommandReceipt(r.Context(), ns, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	raw, _ := json.Marshal(v)
	if len(raw) > maxJSONBodyBytes {
		receipt, err := commands.CommitReceipt(json.RawMessage(v.ResponseJSON))
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, 200, receipt)
		return
	}
	writeJSON(w, 200, v)
}

func (h *Handler) listAvailableWork(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	limit, err := queryLimit(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	v, err := h.service.ListAvailableWork(r.Context(), scope, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, v)
}
