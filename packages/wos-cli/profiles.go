package woscli

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"net/url"
	"regexp"
	"strings"
)

type ProjectV2 struct {
	SchemaVersion int                 `json:"schema_version"`
	Kind          string              `json:"kind"`
	Name          string              `json:"name"`
	Connection    ProjectConnectionV2 `json:"connection"`
	Scope         WorkspaceScope      `json:"scope"`
	Workspace     ProjectLayoutV2     `json:"workspace"`
}
type ProjectConnectionV2 struct {
	ServerURL        string `json:"server_url"`
	ExpectedServerID d.ID   `json:"expected_server_id"`
}
type ProjectLayoutV2 struct {
	ProfilesDirectory string `json:"profiles_directory"`
}
type ProfileBindingV2 struct {
	ServerID     d.ID              `json:"server_id"`
	ServerOrigin string            `json:"server_origin"`
	NamespaceID  d.ID              `json:"namespace_id"`
	PrincipalID  string            `json:"principal_id"`
	CredentialID d.ID              `json:"credential_id"`
	IssuerKeys   []ProfileIssuerV2 `json:"issuer_keys"`
}
type ProfileIssuerV2 struct {
	KeyID       d.ID   `json:"key_id"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}
type ProfileAuthenticationV2 struct {
	CredentialRef string `json:"credential_ref"`
}
type ProfileSigningV2 struct {
	KeyID                d.ID   `json:"key_id"`
	PrivateKeyRef        string `json:"private_key_ref"`
	PublicKeyFingerprint string `json:"public_key_fingerprint"`
}
type ProfileLocalV2 struct {
	SchemaVersion     int                  `json:"schema_version"`
	PendingMAC        string               `json:"pending_mac"`
	BindingMAC        string               `json:"binding_mac"`
	PendingOperations []PendingOperationV2 `json:"pending_operations"`
}
type ProfileV2 struct {
	SchemaVersion  int                     `json:"schema_version"`
	Kind           string                  `json:"kind"`
	Name           string                  `json:"name"`
	Binding        ProfileBindingV2        `json:"binding"`
	Authentication ProfileAuthenticationV2 `json:"authentication"`
	Signing        ProfileSigningV2        `json:"signing"`
	Lease          LeaseConfig             `json:"lease"`
	Output         OutputConfig            `json:"output"`
	Local          ProfileLocalV2          `json:"_local"`
}

// Pending operations are bounded technical state, never an append-only history.
// Their payload is frozen canonical bytes, not a shell instruction.
type ProfilePendingScopeV2 struct {
	NamespaceID d.ID `json:"namespace_id"`
	OutcomeID   d.ID `json:"outcome_id,omitempty"`
}
type PendingOperationV2 struct {
	ID             d.ID                  `json:"id"`
	Operation      string                `json:"operation"`
	State          string                `json:"state"`
	Scope          ProfilePendingScopeV2 `json:"scope"`
	IdempotencyKey string                `json:"idempotency_key"`
	Payload        string                `json:"payload"`
	PayloadDigest  string                `json:"payload_digest"`
	ContractID     *d.ID                 `json:"contract_id,omitempty"`
	Response       string                `json:"response,omitempty"`
}

func validProfileName(name string) bool {
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`).MatchString(name) {
		return false
	}
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL", "CLOCK$":
		return false
	}
	upper := strings.ToUpper(name)
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
		return false
	}
	return true
}
func exactOrigin(raw string) (string, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("server must be an exact HTTP(S) origin")
	}
	if u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return "", fmt.Errorf("plain HTTP is limited to explicit loopback development")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}
func (p ProjectV2) Validate() error {
	if p.SchemaVersion != 2 || p.Kind != "WOSProject" || p.Name == "" || len(p.Name) > 256 || p.Workspace.ProfilesDirectory != "profiles" {
		return fmt.Errorf("invalid project schema 2")
	}
	origin, e := exactOrigin(p.Connection.ServerURL)
	if e != nil || origin != p.Connection.ServerURL {
		return fmt.Errorf("project origin must be canonical")
	}
	if p.Connection.ExpectedServerID.Validate() != nil || p.Scope.NamespaceID.Validate() != nil || p.Scope.DefaultOutcomeID.Validate() != nil {
		return fmt.Errorf("invalid project scope or persistent server ID")
	}
	return nil
}
func (p ProfileV2) Validate() error {
	if p.SchemaVersion != 2 || p.Kind != "WOSProfile" || !validProfileName(p.Name) || p.Local.SchemaVersion != 1 || len(p.Local.PendingOperations) > 10 {
		return fmt.Errorf("invalid profile schema/name or pending limit")
	}
	origin, e := exactOrigin(p.Binding.ServerOrigin)
	if e != nil || origin != p.Binding.ServerOrigin {
		return fmt.Errorf("invalid profile origin")
	}
	if p.Binding.ServerID.Validate() != nil || p.Binding.NamespaceID.Validate() != nil || p.Binding.CredentialID.Validate() != nil || p.Signing.KeyID.Validate() != nil || p.Binding.PrincipalID == "" || len(p.Binding.PrincipalID) > 256 {
		return fmt.Errorf("invalid profile identity")
	}
	if !validSecretReference(p.Authentication.CredentialRef) || !validSecretReference(p.Signing.PrivateKeyRef) || !d.ValidSignedDigest(p.Signing.PublicKeyFingerprint) {
		return fmt.Errorf("profile must contain valid secret references and key fingerprint")
	}
	if len(p.Binding.IssuerKeys) < 1 || len(p.Binding.IssuerKeys) > 10 {
		return fmt.Errorf("profile requires bounded pinned issuer keys")
	}
	seen := map[d.ID]bool{}
	for _, k := range p.Binding.IssuerKeys {
		if k.KeyID.Validate() != nil || seen[k.KeyID] || !d.ValidSignedDigest(k.Fingerprint) {
			return fmt.Errorf("invalid pinned issuer")
		}
		seen[k.KeyID] = true
		public, e := base64.StdEncoding.Strict().DecodeString(k.PublicKey)
		if e != nil || len(public) != ed25519.PublicKeySize || base64.StdEncoding.EncodeToString(public) != k.PublicKey || signing.Digest(public) != k.Fingerprint {
			return fmt.Errorf("invalid pinned issuer public key/fingerprint")
		}

	}
	if p.Lease.RequestedTTLSeconds < 30 || p.Lease.RequestedTTLSeconds > 3600 || (p.Output.DefaultFormat != "json" && p.Output.DefaultFormat != "text") {
		return fmt.Errorf("invalid signed profile lease/output")
	}
	for _, pending := range p.Local.PendingOperations {
		if pending.ID.Validate() != nil || pending.Scope.NamespaceID != p.Binding.NamespaceID || pending.Scope.NamespaceID.Validate() != nil || !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,99}$`).MatchString(pending.Operation) {
			return fmt.Errorf("invalid pending operation binding")
		}
		if pending.Scope.OutcomeID != "" && pending.Scope.OutcomeID.Validate() != nil {
			return fmt.Errorf("invalid pending Outcome")
		}
		if d.ValidateIdempotencyKey(pending.IdempotencyKey) != nil || !validTechnicalV2Blob(pending.Payload) {
			return fmt.Errorf("invalid pending operation payload/key")
		}
		payload, _ := base64.StdEncoding.DecodeString(pending.Payload)
		if len(payload) == 0 || signing.Digest(payload) != pending.PayloadDigest {
			return fmt.Errorf("pending payload digest differs")
		}
		switch pending.State {
		case "prepared", "sent_unknown", "accepted_unmaterialized", "prepared_signed", "accepted_confirmed", "cleanup_pending", "rejected":
		default:
			return fmt.Errorf("invalid pending state")
		}
		if pending.Response != "" && !validTechnicalV2Blob(pending.Response) {
			return fmt.Errorf("invalid pending response")
		}
	}
	return nil
}
func (p ProfileV2) bindingMessage() ([]byte, error) {
	// Mutable drafts/pending metadata and harmless presentation preferences are
	// excluded. Destination, identity, pins and secret references are authenticated.
	return signing.Canonical(struct {
		SchemaVersion  int                     `json:"schema_version"`
		Name           string                  `json:"name"`
		Binding        ProfileBindingV2        `json:"binding"`
		Authentication ProfileAuthenticationV2 `json:"authentication"`
		Signing        ProfileSigningV2        `json:"signing"`
	}{2, p.Name, p.Binding, p.Authentication, p.Signing})
}
func (p *ProfileV2) SealBinding(token []byte) error {
	if len(token) < 32 {
		return fmt.Errorf("credential unavailable")
	}
	raw, e := p.bindingMessage()
	if e != nil {
		return e
	}
	mac := hmac.New(sha256.New, token)
	mac.Write([]byte("WOS local profile destination v2\x00"))
	mac.Write(raw)
	p.Local.BindingMAC = hex.EncodeToString(mac.Sum(nil))
	pending, e := json.Marshal(struct {
		Name       string               `json:"name"`
		BindingMAC string               `json:"binding_mac"`
		Operations []PendingOperationV2 `json:"operations"`
	}{p.Name, p.Local.BindingMAC, p.Local.PendingOperations})
	if e != nil {
		return e
	}
	journal := hmac.New(sha256.New, token)
	journal.Write([]byte("WOS local profile pending v2\x00"))
	journal.Write(pending)
	p.Local.PendingMAC = hex.EncodeToString(journal.Sum(nil))
	return nil
}
func (p ProfileV2) VerifyBinding(token []byte) error {
	expected, e := hex.DecodeString(p.Local.BindingMAC)
	if e != nil || len(expected) != sha256.Size {
		return fmt.Errorf("profile binding proof absent; explicit onboarding required")
	}
	copy := p
	if e = copy.SealBinding(token); e != nil {
		return e
	}
	actual, _ := hex.DecodeString(copy.Local.BindingMAC)
	if !hmac.Equal(expected, actual) {
		return fmt.Errorf("profile destination/identity/secret reference changed; restore or explicitly onboard a new profile")
	}
	pendingExpected, e := hex.DecodeString(p.Local.PendingMAC)
	pendingActual, _ := hex.DecodeString(copy.Local.PendingMAC)
	if e != nil || len(pendingExpected) != sha256.Size || !hmac.Equal(pendingExpected, pendingActual) {
		return fmt.Errorf("profile pending intent changed; preserve it for explicit reconciliation")
	}
	return nil
}
