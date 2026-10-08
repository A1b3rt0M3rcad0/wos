package httptransport

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"reflect"
)

func (h *Handler) signingAdministration(w http.ResponseWriter, r *http.Request) {
	var body json.RawMessage
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	raw, err := commands.Normalize(body, reflect.TypeFor[application.SigningSecurityIntent]())
	if err != nil {
		writeError(w, r, err)
		return
	}
	var intent application.SigningSecurityIntent
	if err = commands.Decode(raw, &intent); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.SigningMutation(r.Context(), r.Header.Get("Idempotency-Key"), intent)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) signingIdentity(w http.ResponseWriter, r *http.Request) {
	var enrollment *domain.ID
	if value := r.URL.Query().Get("enrollment_id"); value != "" {
		id, err := domain.ParseID(value)
		if err != nil {
			writeError(w, r, err)
			return
		}
		enrollment = &id
	}
	result, err := h.security.SigningIdentity(r.Context(), enrollment)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}
