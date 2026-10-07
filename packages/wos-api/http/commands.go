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
		var receipt map[string]json.RawMessage
		json.Unmarshal(raw, &receipt)
		writeJSON(w, http.StatusOK, map[string]any{"command_id": receipt["command_id"], "outcome_revision": receipt["outcome_revision"], "idempotent_replay": receipt["idempotent_replay"], "result_omitted": true})
		return
	}
	writeJSON(w, http.StatusOK, result)
}
