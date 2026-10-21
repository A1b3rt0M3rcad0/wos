package httptransport

import (
	"encoding/json"
	"net/http"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (h *Handler) getEntityWorkspaceView(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var query application.WorkspaceViewQuery
	if err = decodeJSON(w, r, &query); err != nil {
		writeError(w, r, err)
		return
	}
	value, err := h.service.GetEntityWorkspaceView(r.Context(), scope, query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeWorkspaceRead(w, r, value)
}

func (h *Handler) getWorkspaceCards(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var query application.WorkspaceCardsQuery
	if err = decodeJSON(w, r, &query); err != nil {
		writeError(w, r, err)
		return
	}
	value, err := h.service.GetWorkspaceCards(r.Context(), scope, query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeWorkspaceRead(w, r, value)
}

func (h *Handler) getWorkspaceTimeline(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	values := r.URL.Query()
	query := application.WorkspaceTimelineQuery{Cursor: values.Get("cursor")}
	query.Limit, err = queryLimit(r)
	if err != nil || values.Has("limit") && query.Limit < 1 {
		writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "timeline limit must be between 1 and 20"))
		return
	}
	if values.Has("kind") || values.Has("entity_id") {
		id, err := domain.ParseID(values.Get("entity_id"))
		if err != nil {
			writeError(w, r, err)
			return
		}
		query.SubjectRef = &domain.EntityRef{Scope: scope, Kind: domain.EntityKind(values.Get("kind")), ID: id}
	}
	if values.Has("criterion_id") {
		id, err := domain.ParseID(values.Get("criterion_id"))
		if err != nil {
			writeError(w, r, err)
			return
		}
		query.CriterionID = &id
	}
	if values.Has("include_descendants") {
		switch values.Get("include_descendants") {
		case "true":
			query.IncludeDescendants = true
		case "false":
		default:
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "include_descendants must be true or false"))
			return
		}
	}
	value, err := h.service.GetWorkspaceTimeline(r.Context(), scope, query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeWorkspaceRead(w, r, value)
}

func (h *Handler) getEntityNote(w http.ResponseWriter, r *http.Request) {
	scope, err := parseScope(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "note_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	value, err := h.service.GetEntityNote(r.Context(), scope, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeWorkspaceRead(w, r, value)
}

func writeWorkspaceRead(w http.ResponseWriter, r *http.Request, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if len(raw) > commands.MaxPayloadBytes {
		writeError(w, r, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "workspace response exceeds 256 KiB; request a smaller page"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, json.RawMessage(raw))
}
