package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// Called under the name's stable profile lock. Existing OS secret entries are
// reused after interrupted provisioning and are never silently overwritten.
func prepareGeneratedProfileV2(ctx context.Context, w *Workspace, o options, name string, resolver SecretResolver, input io.Reader) (ProfileV2, error) {
	var profile ProfileV2
	project, e := w.LoadProjectV2()
	if e != nil {
		return profile, e
	}
	generate := o.values["generate-signing-key"] == "true"
	importKey := o.values["signing-key-stdin"] == "true"
	if generate == importKey || o.values["enrollment"] == "" || importKey && o.values["token-stdin"] == "true" {
		return profile, usage("select --generate-signing-key or --signing-key-stdin with an authorized --enrollment; token and key cannot share stdin")
	}
	if o.values["server"] != project.Connection.ServerURL || o.values["server-id"] != project.Connection.ExpectedServerID.String() || !d.ValidSignedDigest(o.values["issuer-fingerprint"]) {
		return profile, usage("explicit --server, --server-id and approved --issuer-fingerprint required")
	}
	enrollment, e := d.ParseID(o.values["enrollment"])
	if e != nil {
		return profile, e
	}
	apiRef := o.values["credential-ref"]
	var token []byte
	if o.values["token-stdin"] == "true" {
		if apiRef != "" && apiRef != "keyring:wos/"+name+"/api" {
			return profile, usage("--token-stdin provisions only the selected profile OS keyring entry")
		}
		apiRef = "keyring:wos/" + name + "/api"
		token, e = io.ReadAll(io.LimitReader(input, 8193))
		if e != nil {
			return profile, fmt.Errorf("protected token input failed")
		}
		token = bytes.TrimSuffix(bytes.TrimSuffix(token, []byte("\n")), []byte("\r"))
	} else {
		if apiRef == "" {
			return profile, usage("use --token-stdin or explicit --credential-ref")
		}
		token, e = resolver.Read(apiRef)
		if e != nil {
			return profile, e
		}
	}
	defer clear(token)
	if len(token) < 32 || len(token) > 8192 || bytes.ContainsAny(token, "\r\n\x00") {
		return profile, fmt.Errorf("invalid protected credential input")
	}
	client, e := sdk.New(project.Connection.ServerURL, string(token), nil)
	if e != nil {
		return profile, e
	}
	identity, e := client.SigningIdentity(ctx, &enrollment)
	if e != nil {
		return profile, e
	}
	if identity.ServerID != project.Connection.ExpectedServerID.String() || identity.NamespaceID != project.Scope.NamespaceID || identity.Enrollment == nil || identity.Enrollment.Consumed || identity.Enrollment.ID != enrollment {
		return profile, fmt.Errorf("authorized enrollment/persistent identity differs")
	}
	trust, e := client.GetSignedTrust(ctx, project.Scope.NamespaceID)
	if e != nil {
		return profile, e
	}
	if trust.Server == nil || trust.Server.ID.String() != identity.ServerID || trust.Server.Fingerprint != o.values["issuer-fingerprint"] {
		return profile, fmt.Errorf("server issuer differs from explicitly approved fingerprint")
	}
	signingRef := o.values["private-key-ref"]
	if signingRef == "" {
		signingRef = "keyring:wos/" + name + "/signing"
	}
	if signingRef != "keyring:wos/"+name+"/signing" {
		return profile, usage("generated keys are stored only in the selected profile OS keyring")
	}
	if resolver.Keyring == nil {
		return profile, fmt.Errorf("OS keyring unavailable; use explicit preprovisioned env/mounted references")
	}
	var imported []byte
	if importKey {
		inputBytes, e := io.ReadAll(io.LimitReader(input, 8193))
		if e != nil {
			return profile, fmt.Errorf("protected signing key input failed")
		}
		defer clear(inputBytes)
		inputBytes = bytes.TrimSuffix(bytes.TrimSuffix(inputBytes, []byte("\n")), []byte("\r"))
		imported, e = base64.StdEncoding.Strict().DecodeString(string(inputBytes))
		if e != nil || len(imported) != 32 || base64.StdEncoding.EncodeToString(imported) != string(inputBytes) {
			clear(imported)
			return profile, fmt.Errorf("protected signing input must be a canonical base64 Ed25519 seed")
		}
		defer clear(imported)
	}
	secret, e := resolver.Keyring.Get("wos", name+"/signing")
	var seed []byte
	if e == nil {
		seed, e = base64.StdEncoding.Strict().DecodeString(secret)
		if e != nil || len(seed) != 32 || base64.StdEncoding.EncodeToString(seed) != secret {
			clear(seed)
			return profile, fmt.Errorf("existing signing entry is invalid; preserved for explicit recovery")
		}
		if importKey && !bytes.Equal(seed, imported) {
			clear(seed)
			return profile, fmt.Errorf("existing signing entry differs; preserved")
		}
	} else {
		if !keyringMissingV2(e) {
			return profile, fmt.Errorf("OS keyring unavailable; no plaintext fallback")
		}
		seed = make([]byte, 32)
		if importKey {
			copy(seed, imported)
		} else if _, e = rand.Read(seed); e != nil {
			return profile, e
		}
		if e = resolver.Store(signingRef, []byte(base64.StdEncoding.EncodeToString(seed))); e != nil {
			clear(seed)
			return profile, e
		}
	}
	defer clear(seed)
	private := ed25519.NewKeyFromSeed(seed)
	defer clear(private)
	fingerprint := signing.Digest(private.Public().(ed25519.PublicKey))
	challenge := identity.Enrollment
	if challenge.CredentialID != identity.CredentialID || challenge.PrincipalID != identity.PrincipalID || challenge.Purpose != "agent" || challenge.ExpectedFingerprint != "" && challenge.ExpectedFingerprint != fingerprint {
		return profile, fmt.Errorf("enrollment fingerprint/subject differs; generated entry preserved")
	}
	if strings.HasPrefix(apiRef, "keyring:") && o.values["token-stdin"] == "true" {
		existing, e := resolver.Keyring.Get("wos", name+"/api")
		if e == nil {
			if !bytes.Equal([]byte(existing), token) {
				return profile, fmt.Errorf("existing credential entry differs; preserved")
			}
		} else {
			if !keyringMissingV2(e) {
				return profile, fmt.Errorf("OS keyring credential unavailable")
			}
			if e = resolver.Store(apiRef, token); e != nil {
				return profile, e
			}
		}
	}
	profile = ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: name, Binding: ProfileBindingV2{ServerID: project.Connection.ExpectedServerID, ServerOrigin: project.Connection.ServerURL, NamespaceID: identity.NamespaceID, PrincipalID: identity.PrincipalID, CredentialID: identity.CredentialID, IssuerKeys: []ProfileIssuerV2{{KeyID: trust.Server.IssuerKeyID, PublicKey: trust.Server.PublicKey, Fingerprint: trust.Server.Fingerprint}}}, Authentication: ProfileAuthenticationV2{apiRef}, Signing: ProfileSigningV2{KeyID: challenge.KeyID, PrivateKeyRef: signingRef, PublicKeyFingerprint: fingerprint}, Lease: LeaseConfig{300}, Output: OutputConfig{"json"}, Local: ProfileLocalV2{SchemaVersion: 1, PendingOperations: []PendingOperationV2{}}}
	return profile, profile.Validate()
}
