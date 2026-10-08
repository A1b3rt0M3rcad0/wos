package httptransport

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"net/http"
)

func (h *Handler) commandCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"schema_version": 1, "commands": h.commands.Descriptors()})
}
func (h *Handler) executeCommand(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Command json.RawMessage `json:"command"`
	}
	if err := decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.commands.Execute(r.Context(), r.PathValue("command"), r.Header.Get("Idempotency-Key"), r.Header.Get("X-Correlation-ID"), b.Command)
	if err != nil {
		writeError(w, r, err)
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if len(raw) > commands.MaxPayloadBytes {
		receipt, err := commands.CommitReceipt(result)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, receipt)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
