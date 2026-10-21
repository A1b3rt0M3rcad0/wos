package application

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// CredentialIssuanceSelection describes the public, non-secret choices made
// when a bearer credential is issued. The selected template is materialized
// into concrete permissions at issuance time; it is never evaluated again.
type CredentialIssuanceSelection struct {
	NamespaceID          domain.ID
	CredentialID         domain.ID
	PrincipalID          string
	Actor                domain.ActorRef
	ExpiresAt            time.Time
	Name                 string
	Description          string
	TemplateID           ports.PermissionTemplateID
	TemplateVersion      uint32
	EditedPermissions    []ports.Permission
	AllowedOutcomeIDs    []domain.ID
	AllowedSigningKeyIDs []domain.ID
	CreatedByPrincipalID string
	CreatedByActor       domain.ActorRef
}

// CredentialIssuanceMetadata is safe to persist and return in administrative
// projections. It deliberately has no bearer token, digest, or private key.
type CredentialIssuanceMetadata = ports.CredentialMetadata

// CredentialIssuancePreparation is the validated, detached policy and
// provenance that an adapter must persist atomically with the credential.
type CredentialIssuancePreparation struct {
	Policy   domain.CredentialPolicy
	Metadata CredentialIssuanceMetadata
	Preview  PermissionTemplatePreview
}

// PrepareCredentialIssuance freezes a template/custom permission list and
// intersects it with the target Principal's current Namespace grant. Storage
// must recheck that the prepared permission set remains within the grant in
// the same transaction that inserts the credential and policy.
func PrepareCredentialIssuance(selection CredentialIssuanceSelection, grant ports.NamespaceGrant, now time.Time) (CredentialIssuancePreparation, error) {
	var result CredentialIssuancePreparation
	if selection.NamespaceID.Validate() != nil || selection.CredentialID.Validate() != nil || grant.NamespaceID != selection.NamespaceID || grant.PrincipalID != selection.PrincipalID || strings.TrimSpace(selection.PrincipalID) == "" || len(selection.PrincipalID) > 256 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential issuance scope or Principal is invalid")
	}
	if err := selection.Actor.Validate(); err != nil {
		return result, err
	}
	if strings.TrimSpace(selection.CreatedByPrincipalID) == "" {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential issuer Principal is required")
	}
	if err := selection.CreatedByActor.Validate(); err != nil {
		return result, err
	}
	if now.IsZero() || !selection.ExpiresAt.After(now) || selection.ExpiresAt.After(now.Add(366*24*time.Hour)) {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential expiration must be within one year")
	}
	name := strings.TrimSpace(selection.Name)
	if name == "" {
		// Preserve the legacy issuance endpoint while giving every new key a
		// useful administrative label until the caller supplies a custom name.
		name = "Issued credential"
	}
	description := strings.TrimSpace(selection.Description)
	if utf8.RuneCountInString(name) > 100 || utf8.RuneCountInString(description) > 1000 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential name is limited to 100 characters and description to 1000")
	}
	if len(selection.AllowedOutcomeIDs) > 100 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential Outcome scope exceeds 100 entries")
	}
	allowedOutcomes := append([]domain.ID(nil), selection.AllowedOutcomeIDs...)
	seenOutcomes := make(map[domain.ID]struct{}, len(allowedOutcomes))
	for _, id := range allowedOutcomes {
		if id.Validate() != nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential Outcome scope contains an invalid ID")
		}
		if _, exists := seenOutcomes[id]; exists {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential Outcome scope contains a duplicate ID")
		}
		seenOutcomes[id] = struct{}{}
	}
	allowedOutcomes = sortedIDCopy(allowedOutcomes)
	if len(selection.AllowedSigningKeyIDs) > 100 {
		return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential signing-key scope exceeds 100 entries")
	}
	allowedKeys := sortedIDCopy(selection.AllowedSigningKeyIDs)
	seenKeys := make(map[domain.ID]struct{}, len(allowedKeys))
	for _, id := range allowedKeys {
		if id.Validate() != nil {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential signing-key scope contains an invalid ID")
		}
		if _, exists := seenKeys[id]; exists {
			return result, domain.NewError(domain.ErrorCodeInvalidArgument, "credential signing-key scope contains a duplicate ID")
		}
		seenKeys[id] = struct{}{}
	}

	// An omitted role preserves the existing endpoint contract by freezing the
	// Principal grant visible at issuance. It does not follow later grant growth.
	permissionSelection := PermissionTemplateSelection{
		TemplateID:      selection.TemplateID,
		TemplateVersion: selection.TemplateVersion,
	}
	if selection.EditedPermissions != nil {
		permissionSelection.EditedPermissions = append([]ports.Permission{}, selection.EditedPermissions...)
	}
	if selection.TemplateID == "" && selection.TemplateVersion == 0 && selection.EditedPermissions == nil {
		permissionSelection.EditedPermissions = append([]ports.Permission{}, grant.Permissions...)
	}
	preview, err := PreviewPermissionTemplate(PermissionTemplatePreviewInput{
		Selection:                 permissionSelection,
		NamespaceGrantPermissions: grant.Permissions,
	})
	if err != nil {
		return result, err
	}
	permitted := make([]string, 0, len(preview.EffectivePermissions))
	for _, p := range preview.EffectivePermissions {
		permitted = append(permitted, string(p))
	}
	policy := domain.CredentialPolicy{
		NamespaceID:              selection.NamespaceID,
		CredentialID:             selection.CredentialID,
		PrincipalID:              selection.PrincipalID,
		Version:                  1,
		PermittedOperations:      permitted,
		AcceptanceFloor:          domain.AcceptanceDirect,
		AllowedOutcomeIDs:        allowedOutcomes,
		AllowedSigningKeyIDs:     allowedKeys,
		MaxActiveWorkContracts:   3,
		MaxActiveReviewContracts: 1,
		UpdatedAt:                now.UTC(),
	}
	if err := policy.Validate(); err != nil {
		return result, err
	}
	metadata := CredentialIssuanceMetadata{
		NamespaceID:          selection.NamespaceID,
		CredentialID:         selection.CredentialID,
		Name:                 name,
		Description:          description,
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.UTC(),
		MetadataRevision:     1,
		CreatedByPrincipalID: selection.CreatedByPrincipalID,
		CreatedByActor:       selection.CreatedByActor,
		RoleTemplateID:       preview.Materialization.TemplateID,
		RoleTemplateVersion:  preview.Materialization.TemplateVersion,
		PolicyRevision:       policy.Version,
	}
	return CredentialIssuancePreparation{Policy: policy, Metadata: metadata, Preview: preview}, nil
}

func sortedIDCopy(values []domain.ID) []domain.ID {
	result := append([]domain.ID(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
