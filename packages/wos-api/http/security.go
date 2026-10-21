package httptransport

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	v, err := h.security.Namespaces(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": v})
}
func (h *Handler) setGrant(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var b struct {
		Permissions     []ports.Permission `json:"permissions"`
		ExpectedVersion domain.Version     `json:"expected_namespace_version"`
		Reason          string             `json:"reason"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	result, _, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: b.ExpectedVersion, Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: r.PathValue("principal_id"), Permissions: b.Permissions}, Reason: b.Reason})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *Handler) issueCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var b struct {
		PrincipalID          string                     `json:"principal_id"`
		Actor                domain.ActorRef            `json:"actor_ref"`
		ExpiresAt            time.Time                  `json:"expires_at"`
		ExpectedVersion      domain.Version             `json:"expected_namespace_version"`
		CredentialName       string                     `json:"credential_name,omitempty"`
		Description          string                     `json:"credential_description,omitempty"`
		RoleTemplateID       ports.PermissionTemplateID `json:"role_template_id,omitempty"`
		RoleTemplateVersion  uint32                     `json:"role_template_version,omitempty"`
		Permissions          []ports.Permission         `json:"permissions,omitempty"`
		AllowedOutcomeIDs    []domain.ID                `json:"allowed_outcome_ids,omitempty"`
		AllowedSigningKeyIDs []domain.ID                `json:"allowed_signing_key_ids,omitempty"`
		Reason               string                     `json:"reason"`
	}
	if err = decodeJSON(w, r, &b); err != nil {
		writeError(w, r, err)
		return
	}
	result, token, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: b.ExpectedVersion, Operation: "issue_credential", Credential: &ports.Credential{NamespaceID: ns, PrincipalID: b.PrincipalID, Actor: b.Actor, ExpiresAt: b.ExpiresAt}, CredentialName: b.CredentialName, CredentialDescription: b.Description, RoleTemplateID: b.RoleTemplateID, RoleTemplateVersion: b.RoleTemplateVersion, CredentialPermissions: b.Permissions, AllowedOutcomeIDs: b.AllowedOutcomeIDs, AllowedSigningKeyIDs: b.AllowedSigningKeyIDs, Reason: b.Reason})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"result": result, "token": token})
}
func (h *Handler) revokeCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedVersion domain.Version `json:"expected_namespace_version"`
		Reason          string         `json:"reason"`
	}
	if err = decodeJSON(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, _, err := h.security.AdministrativeMutation(r.Context(), r.Header.Get("Idempotency-Key"), ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: body.ExpectedVersion, Operation: "revoke_credential", CredentialID: id, Reason: body.Reason})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) previewCredentialRevocation(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.PreviewCredentialRevocation(r.Context(), ns, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) securityAdministration(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.AdministrativeSnapshot(r.Context(), ns)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	query := ports.CredentialManagementQuery{PrincipalID: r.URL.Query().Get("principal_id"), Status: r.URL.Query().Get("status"), ActorKind: domain.ActorKind(r.URL.Query().Get("actor_kind")), ActorID: r.URL.Query().Get("actor_id")}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		query.Limit, err = strconv.Atoi(raw)
		if err != nil {
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "credential limit must be an integer"))
			return
		}
	}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		query.Cursor, err = domain.ParseID(raw)
		if err != nil {
			writeError(w, r, domain.NewError(domain.ErrorCodeInvalidArgument, "credential cursor is invalid"))
			return
		}
	}
	result, err := h.security.ListCredentials(r.Context(), ns, query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.GetCredential(r.Context(), ns, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) updateCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedNamespaceVersion domain.Version           `json:"expected_namespace_version"`
		ExpectedMetadataRevision domain.Version           `json:"expected_metadata_revision"`
		ExpectedPolicyRevision   domain.Version           `json:"expected_policy_revision"`
		Name                     string                   `json:"name"`
		Description              string                   `json:"description"`
		ExpiresAt                *time.Time               `json:"expires_at,omitempty"`
		Policy                   *domain.CredentialPolicy `json:"credential_policy,omitempty"`
		Reason                   string                   `json:"reason"`
	}
	if err = decodeJSON(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.security.UpdateCredential(r.Context(), r.Header.Get("Idempotency-Key"), application.CredentialUpdateIntent{NamespaceID: ns, CredentialID: id, ExpectedNamespaceVersion: body.ExpectedNamespaceVersion, ExpectedMetadataRevision: body.ExpectedMetadataRevision, ExpectedPolicyRevision: body.ExpectedPolicyRevision, Name: body.Name, Description: body.Description, ExpiresAt: body.ExpiresAt, Policy: body.Policy, Reason: body.Reason})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) rotateCredential(w http.ResponseWriter, r *http.Request) {
	ns, err := parsePathID(r, "namespace_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	id, err := parsePathID(r, "credential_id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	var body struct {
		ExpectedNamespaceVersion domain.Version `json:"expected_namespace_version"`
		ExpectedMetadataRevision domain.Version `json:"expected_metadata_revision"`
		ExpectedPolicyRevision   domain.Version `json:"expected_policy_revision"`
		ExpiresAt                time.Time      `json:"expires_at"`
		OverlapUntil             time.Time      `json:"overlap_until"`
		Name                     string         `json:"name,omitempty"`
		Description              string         `json:"description,omitempty"`
		Reason                   string         `json:"reason"`
	}
	if err = decodeJSON(w, r, &body); err != nil {
		writeError(w, r, err)
		return
	}
	result, token, err := h.security.RotateCredential(r.Context(), r.Header.Get("Idempotency-Key"), application.CredentialRotationIntent{NamespaceID: ns, CredentialID: id, ExpectedNamespaceVersion: body.ExpectedNamespaceVersion, ExpectedMetadataRevision: body.ExpectedMetadataRevision, ExpectedPolicyRevision: body.ExpectedPolicyRevision, ExpiresAt: body.ExpiresAt, OverlapUntil: body.OverlapUntil, Name: body.Name, Description: body.Description, Reason: body.Reason})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"result": result, "token": token})
}

func (h *Handler) administer(w http.ResponseWriter, r *http.Request) {
	var intent application.AdministrativeIntent
	if err := decodeJSON(w, r, &intent); err != nil {
		writeError(w, r, err)
		return
	}
	result, token, err := h.security.Administer(r.Context(), r.Header.Get("Idempotency-Key"), intent)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"result": result, "token": token})
}
