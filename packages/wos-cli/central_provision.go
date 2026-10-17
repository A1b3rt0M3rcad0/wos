package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// Protected env/mounted references are an explicit alternative to an OS keyring,
// not a plaintext fallback. Provisioning never exports their contents and the
// unchanged onboarding path freezes/signs/reconciles enrollment afterwards.
func prepareReferencedProfileV2(ctx context.Context, w *Workspace, o options, name string, resolver SecretResolver) (ProfileV2, error) {
	var profile ProfileV2
	project, e := w.LoadProjectV2()
	if e != nil {
		return profile, e
	}
	if o.values["token-stdin"] == "true" || o.values["generate-signing-key"] == "true" || o.values["signing-key-stdin"] == "true" {
		return profile, usage("protected reference provisioning cannot also generate/import a signing key or token")
	}
	if o.values["server"] != project.Connection.ServerURL || o.values["server-id"] != project.Connection.ExpectedServerID.String() || !d.ValidSignedDigest(o.values["issuer-fingerprint"]) {
		return profile, usage("explicit --server, --server-id and approved --issuer-fingerprint required")
	}
	credentialRef := o.values["credential-ref"]
	privateRef := o.values["private-key-ref"]
	if !validSecretReference(credentialRef) || !validSecretReference(privateRef) {
		return profile, usage("supply --credential-ref and --private-key-ref as protected env/keyring/mounted references")
	}
	token, e := resolver.Read(credentialRef)
	if e != nil {
		return profile, e
	}
	defer clear(token)
	if len(token) < 32 || len(token) > 8192 || bytes.ContainsAny(token, "\r\n\x00") {
		return profile, fmt.Errorf("invalid protected credential input")
	}
	secret, e := resolver.Read(privateRef)
	if e != nil {
		return profile, e
	}
	defer clear(secret)
	raw, e := base64.StdEncoding.Strict().DecodeString(string(secret))
	defer clear(raw)
	if e != nil || base64.StdEncoding.EncodeToString(raw) != string(secret) || (len(raw) != ed25519.SeedSize && len(raw) != ed25519.PrivateKeySize) {
		return profile, fmt.Errorf("signing secret must be canonical base64 Ed25519 seed or private key")
	}
	private := ed25519.NewKeyFromSeed(raw[:ed25519.SeedSize])
	defer clear(private)
	if len(raw) == ed25519.PrivateKeySize && !bytes.Equal(private, raw) {
		return profile, fmt.Errorf("private key public half differs")
	}
	fingerprint := signing.Digest(private.Public().(ed25519.PublicKey))
	var enrollment *d.ID
	if o.values["enrollment"] != "" {
		id, e := d.ParseID(o.values["enrollment"])
		if e != nil {
			return profile, e
		}
		enrollment = &id
	}
	client, e := sdk.New(project.Connection.ServerURL, string(token), nil)
	if e != nil {
		return profile, e
	}
	identity, e := client.SigningIdentity(ctx, enrollment)
	if e != nil {
		return profile, e
	}
	if identity.ServerID != project.Connection.ExpectedServerID.String() || identity.TenantID != project.Scope.TenantID {
		return profile, fmt.Errorf("authorized enrollment/persistent identity differs")
	}
	trust, e := client.GetSignedTrust(ctx, project.Scope.TenantID)
	if e != nil {
		return profile, e
	}
	if trust.Server == nil || trust.Server.ID.String() != identity.ServerID || trust.Server.Fingerprint != o.values["issuer-fingerprint"] {
		return profile, fmt.Errorf("server issuer differs from explicitly approved fingerprint")
	}
	var keyID d.ID
	if enrollment != nil {
		challenge := identity.Enrollment
		if challenge == nil || challenge.ID != *enrollment || challenge.Consumed || challenge.CredentialID != identity.CredentialID || challenge.PrincipalID != identity.PrincipalID || challenge.Purpose != "agent" || challenge.ExpectedFingerprint != "" && challenge.ExpectedFingerprint != fingerprint {
			return profile, fmt.Errorf("enrollment fingerprint/subject differs")
		}
		keyID = challenge.KeyID
	} else {
		for _, key := range identity.Keys {
			if key.Status == "active" && key.Purpose == "agent" && key.PrincipalID == identity.PrincipalID && key.Fingerprint == fingerprint {
				if !keyID.IsZero() {
					return profile, fmt.Errorf("registered signing key is ambiguous; use an explicit authorized enrollment")
				}
				keyID = key.ID
			}
		}
		if keyID.IsZero() {
			return profile, fmt.Errorf("selected protected key is not registered; an authorized --enrollment is required")
		}
	}
	profile = ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: name, Binding: ProfileBindingV2{ServerID: project.Connection.ExpectedServerID, ServerOrigin: project.Connection.ServerURL, TenantID: identity.TenantID, PrincipalID: identity.PrincipalID, CredentialID: identity.CredentialID, IssuerKeys: []ProfileIssuerV2{{KeyID: trust.Server.IssuerKeyID, PublicKey: trust.Server.PublicKey, Fingerprint: trust.Server.Fingerprint}}}, Authentication: ProfileAuthenticationV2{CredentialRef: credentialRef}, Signing: ProfileSigningV2{KeyID: keyID, PrivateKeyRef: privateRef, PublicKeyFingerprint: fingerprint}, Lease: LeaseConfig{300}, Output: OutputConfig{"json"}, Local: ProfileLocalV2{SchemaVersion: 1, PendingOperations: []PendingOperationV2{}}}
	return profile, profile.Validate()
}
