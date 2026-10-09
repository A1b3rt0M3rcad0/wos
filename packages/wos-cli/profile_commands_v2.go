package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func newLocalIDV2() (d.ID, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	timestamp := uint64(time.Now().UnixMilli())
	for n := 5; n >= 0; n-- {
		b[n] = byte(timestamp)
		timestamp >>= 8
	}
	b[6] = (b[6] & 15) | 0x70
	b[8] = (b[8] & 63) | 0x80
	h := hex.EncodeToString(b)
	return d.ParseID(h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:])
}
func profilePathV2(name string) string { return filepath.Join(".wos/profiles", name, "profile.yaml") }
func defaultSecretResolverV2(w *Workspace) SecretResolver {
	return SecretResolver{WorkspaceRoot: w.canonicalPath, Keyring: OSKeyring{}}
}

func (w *Workspace) readDirectoryV2(path string) ([]os.DirEntry, error) {
	if e := w.check(path); e != nil {
		return nil, e
	}
	directory, e := w.root.Open(path)
	if e != nil {
		return nil, e
	}
	defer directory.Close()
	info, e := directory.Stat()
	if e != nil || !info.IsDir() {
		return nil, fmt.Errorf("expected a confined directory")
	}
	return directory.ReadDir(-1)
}

// Verify the local immutable destination before constructing a network client.
func loadProfileClientV2(w *Workspace, name string, resolver SecretResolver) (ProfileV2, *sdk.Client, []byte, error) {
	profile, e := w.LoadProfileV2(name)
	if e != nil {
		return profile, nil, nil, e
	}
	token, e := resolver.Read(profile.Authentication.CredentialRef)
	if e != nil {
		return profile, nil, nil, e
	}
	if e = profile.VerifyBinding(token); e != nil {
		clear(token)
		return profile, nil, nil, e
	}
	client, e := sdk.New(profile.Binding.ServerOrigin, string(token), nil)
	if e != nil {
		clear(token)
		return profile, nil, nil, e
	}
	return profile, client, token, nil
}
func checkProfileIdentityV2(ctx context.Context, client *sdk.Client, profile ProfileV2, enrollment *d.ID) (a.SigningIdentityView, error) {
	identity, e := client.SigningIdentity(ctx, enrollment)
	if e != nil {
		return identity, e
	}
	if identity.ServerID != profile.Binding.ServerID.String() || identity.NamespaceID != profile.Binding.NamespaceID || identity.PrincipalID != profile.Binding.PrincipalID || identity.CredentialID != profile.Binding.CredentialID {
		return identity, fmt.Errorf("remote identity differs from selected profile")
	}
	trust, e := client.GetSignedTrust(ctx, profile.Binding.NamespaceID)
	if e != nil {
		return identity, e
	}
	if trust.Server == nil || trust.Server.ID != profile.Binding.ServerID {
		return identity, fmt.Errorf("persistent server identity differs")
	}
	matched := false
	for _, pin := range profile.Binding.IssuerKeys {
		if pin.KeyID == trust.Server.IssuerKeyID && pin.PublicKey == trust.Server.PublicKey && pin.Fingerprint == trust.Server.Fingerprint {
			matched = true
		}
	}
	if !matched {
		return identity, fmt.Errorf("current issuer is outside approved profile pins")
	}
	return identity, nil
}
func readProfilePrivateV2(profile ProfileV2, resolver SecretResolver) (ed25519.PrivateKey, error) {
	secret, e := resolver.Read(profile.Signing.PrivateKeyRef)
	if e != nil {
		return nil, e
	}
	defer clear(secret)
	raw, e := base64.StdEncoding.Strict().DecodeString(string(secret))
	if e != nil || base64.StdEncoding.EncodeToString(raw) != string(secret) {
		clear(raw)
		return nil, fmt.Errorf("signing secret must be canonical base64 Ed25519 seed or private key")
	}
	var key ed25519.PrivateKey
	switch len(raw) {
	case ed25519.SeedSize:
		key = ed25519.NewKeyFromSeed(raw)
	case ed25519.PrivateKeySize:
		key = ed25519.NewKeyFromSeed(raw[:32])
		if !bytes.Equal(key, raw) {
			clear(raw)
			clear(key)
			return nil, fmt.Errorf("private key public half differs")
		}
	default:
		clear(raw)
		return nil, fmt.Errorf("invalid signing secret length")
	}
	clear(raw)
	if signing.Digest(key.Public().(ed25519.PublicKey)) != profile.Signing.PublicKeyFingerprint {
		clear(key)
		return nil, fmt.Errorf("selected private key differs from profile fingerprint")
	}
	return key, nil
}
func compactProfileV2(profile ProfileV2) any {
	return map[string]any{"name": profile.Name, "server_id": profile.Binding.ServerID, "server_origin": profile.Binding.ServerOrigin, "namespace_id": profile.Binding.NamespaceID, "principal_id": profile.Binding.PrincipalID, "credential_id": profile.Binding.CredentialID, "signing_key_id": profile.Signing.KeyID, "signing_fingerprint": profile.Signing.PublicKeyFingerprint, "pending_count": len(profile.Local.PendingOperations)}
}

func compactProfileStatusV2(profile ProfileV2, identity a.SigningIdentityView) any {
	var selected any
	for _, key := range identity.Keys {
		if key.ID == profile.Signing.KeyID {
			selected = map[string]any{"key_id": key.ID, "status": key.Status, "fingerprint": key.Fingerprint, "purpose": key.Purpose}
		}
	}
	var restriction any
	if identity.Policy != nil {
		restriction = map[string]any{"acceptance_floor": identity.Policy.AcceptanceFloor, "revision": signing.Decimal(identity.Policy.Version), "max_active_work_contracts": identity.Policy.MaxActiveWorkContracts, "max_active_review_contracts": identity.Policy.MaxActiveReviewContracts, "selected_key_permitted": identity.Policy.PermitsKey(profile.Signing.KeyID)}
	}
	omitted := len(identity.Keys)
	if selected != nil {
		omitted--
	}
	return map[string]any{"profile": compactProfileV2(profile), "selected_key": selected, "credential_restriction": restriction, "keys_truncated": identity.KeysTruncated, "omitted_other_keys": omitted, "authorization": "each operation rechecks current grant, credential policy, scope and authority on the server"}
}
func projectInitializeV2(w *Workspace, o options) (Output, error) {
	result := Output{Operation: "project init"}
	if len(o.args) != 2 || o.args[1] != "init" {
		return result, usage("use project init with explicit destination IDs")
	}
	server, e := exactOrigin(o.values["server"])
	if e != nil {
		return result, e
	}
	serverID, e := d.ParseID(o.values["server-id"])
	if e != nil {
		return result, e
	}
	namespace, e := d.ParseID(o.values["namespace"])
	if e != nil {
		return result, e
	}
	outcome, e := d.ParseID(o.values["outcome"])
	if e != nil {
		return result, e
	}
	name := o.values["name"]
	if name == "" {
		name = filepath.Base(w.canonicalPath)
	}
	project := ProjectV2{SchemaVersion: 2, Kind: "WOSProject", Name: name, Connection: ProjectConnectionV2{server, serverID}, Scope: WorkspaceScope{namespace, outcome}, Workspace: ProjectLayoutV2{"profiles"}}
	if e = project.Validate(); e != nil {
		return result, e
	}
	if e = w.CreateV2(".wos/project.yaml", project); e != nil {
		return result, e
	}
	result.Data = project
	return result, nil
}

func profileCommandV2(ctx context.Context, w *Workspace, o options, resolver SecretResolver) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " ")}
	if len(o.args) < 2 {
		return result, usage("profile list|create|inspect|recover|remove")
	}
	if o.args[1] == "list" {
		if len(o.args) != 2 {
			return result, usage("profile list takes no profile argument")
		}
		names, e := w.ProfileNamesV2()
		result.Data = names
		return result, e
	}
	name := o.values["profile"]
	if len(o.args) == 3 {
		if name != "" && name != o.args[2] {
			return result, usage("profile selectors differ")
		}
		name = o.args[2]
	}
	if len(o.args) > 3 {
		return result, usage("unexpected profile arguments")
	}
	if o.args[1] == "create" {
		return createProfileV2(ctx, w, o, name, resolver)
	}
	selected, e := w.SelectProfileV2(name, os.Getenv("WOS_PROFILE"))
	if e != nil {
		return result, e
	}
	release, e := w.LockV2(ctx, selected, "")
	if e != nil {
		return result, e
	}
	defer release()
	profile, client, token, e := loadProfileClientV2(w, selected, resolver)
	if e != nil {
		return result, e
	}
	defer clear(token)
	switch o.args[1] {
	case "inspect":
		identity, e := checkProfileIdentityV2(ctx, client, profile, nil)
		if e != nil {
			return result, e
		}
		result.Data = compactProfileStatusV2(profile, identity)
		return result, nil
	case "recover":
		return recoverProfileEnrollmentV2(ctx, w, profile, client, token)
	case "remove":
		if len(profile.Local.PendingOperations) != 0 {
			return result, fmt.Errorf("pending intentions must be reconciled before profile removal")
		}
		directory := filepath.Join(".wos/profiles", selected)
		entries, e := w.readDirectoryV2(directory)
		if e != nil {
			return result, e
		}
		for _, entry := range entries {
			if entry.Name() == "profile.yaml" {
				continue
			}
			if entry.Name() != "contract" || !entry.IsDir() {
				return result, fmt.Errorf("unrecognized profile content preserved")
			}
			children, e := w.readDirectoryV2(filepath.Join(directory, "contract"))
			if e != nil {
				return result, e
			}
			if len(children) > 0 {
				return result, fmt.Errorf("local contracts must be confirmed before profile removal")
			}
		}
		if e = w.root.Remove(profilePathV2(selected)); e != nil {
			return result, e
		}
		// Remove only known empty directories. Shared secrets and remote identities
		// remain intact; this operation does not claim server revocation.
		e = w.root.Remove(filepath.Join(directory, "contract"))
		if e != nil && !os.IsNotExist(e) {
			return result, e
		}
		if e = w.root.Remove(directory); e != nil {
			return result, e
		}
		result.Data = map[string]any{"removed": selected, "remote_identity_revoked": false, "secrets_removed": false}
		return result, nil
	default:
		return result, usage("profile list|create|inspect|recover|remove")
	}
}

func createProfileV2(ctx context.Context, w *Workspace, o options, name string, resolver SecretResolver) (Output, error) {
	result := Output{Operation: "profile create"}
	if !validProfileName(name) {
		return result, usage("profile create requires a valid name")
	}
	release, e := w.LockV2(ctx, name, "")
	if e != nil {
		return result, e
	}
	defer release()
	names, e := w.ProfileNamesV2()
	if e != nil {
		return result, e
	}
	for _, existing := range names {
		if strings.EqualFold(existing, name) {
			return result, fmt.Errorf("profile name already reserved")
		}
	}
	var profile ProfileV2
	if o.values["file"] != "" {
		if o.values["generate-signing-key"] == "true" || o.values["token-stdin"] == "true" || o.values["signing-key-stdin"] == "true" {
			return result, usage("proposal references and generated onboarding are separate explicit modes")
		}
		proposal, e := w.ReadV2(o.values["file"])
		if e != nil {
			return result, e
		}
		if e = DecodeV2Document(proposal, &profile); e != nil {
			return result, e
		}
	} else {
		profile, e = prepareGeneratedProfileV2(ctx, w, o, name, resolver, os.Stdin)
		if e != nil {
			return result, e
		}
	}
	if profile.Name != name || len(profile.Local.PendingOperations) != 0 {
		return result, fmt.Errorf("new profile name differs or contains pending operations")
	}
	if e = profile.Validate(); e != nil {
		return result, e
	}
	project, e := w.LoadProjectV2()
	if e != nil {
		return result, e
	}
	if project.Connection.ServerURL != profile.Binding.ServerOrigin || project.Connection.ExpectedServerID != profile.Binding.ServerID || project.Scope.NamespaceID != profile.Binding.NamespaceID {
		return result, fmt.Errorf("project/proposal binding differs")
	}
	// Separate explicit approval is mandatory on first onboarding; project and
	// proposal files alone are insufficient permission to transmit a credential.
	if o.values["server"] != profile.Binding.ServerOrigin || o.values["server-id"] != profile.Binding.ServerID.String() {
		return result, usage("approve exact --server and --server-id before onboarding")
	}
	approved := false
	for _, pin := range profile.Binding.IssuerKeys {
		if pin.Fingerprint == o.values["issuer-fingerprint"] {
			approved = true
		}
	}
	if !approved {
		return result, usage("approve --issuer-fingerprint from the administrator")
	}
	if len(profile.Binding.IssuerKeys) != 1 {
		return result, usage("initial onboarding approves exactly one issuer; trust rotation must be explicit")
	}
	token, e := resolver.Read(profile.Authentication.CredentialRef)
	if e != nil {
		return result, e
	}
	defer clear(token)
	private, e := readProfilePrivateV2(profile, resolver)
	if e != nil {
		return result, e
	}
	defer clear(private)
	client, e := sdk.New(profile.Binding.ServerOrigin, string(token), nil)
	if e != nil {
		return result, e
	}
	var enrollment *d.ID
	if o.values["enrollment"] != "" {
		id, e := d.ParseID(o.values["enrollment"])
		if e != nil {
			return result, e
		}
		enrollment = &id
	}
	identity, e := checkProfileIdentityV2(ctx, client, profile, enrollment)
	if e != nil {
		return result, e
	}
	if enrollment == nil {
		found := false
		for _, key := range identity.Keys {
			if key.ID == profile.Signing.KeyID && key.Status == "active" && key.Fingerprint == profile.Signing.PublicKeyFingerprint && key.Purpose == "agent" && key.PrincipalID == profile.Binding.PrincipalID {
				found = true
			}
		}
		if !found {
			return result, fmt.Errorf("selected key is not registered; an authorized --enrollment is required")
		}
	} else {
		challenge := identity.Enrollment
		if challenge == nil || challenge.ID != *enrollment || challenge.Consumed || challenge.KeyID != profile.Signing.KeyID || challenge.CredentialID != profile.Binding.CredentialID || challenge.PrincipalID != profile.Binding.PrincipalID {
			return result, fmt.Errorf("enrollment differs from proposed key/identity")
		}
		previous := ""
		if challenge.PreviousKeyID != nil {
			previous = challenge.PreviousKeyID.String()
		}
		payload := signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: profile.Binding.ServerID.String(), NamespaceID: profile.Binding.NamespaceID.String(), PrincipalID: profile.Binding.PrincipalID, SignerKeyID: profile.Signing.KeyID.String()}, EnrollmentID: challenge.ID.String(), Nonce: challenge.Nonce, Purpose: challenge.Purpose, PublicKey: base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey)), ExpiresAt: challenge.ExpiresAt.UTC().Format(time.RFC3339Nano), PreviousKeyID: previous}
		proof, e := signing.Sign(signing.KeyEnrollment, payload, profile.Signing.KeyID.String(), private)
		if e != nil {
			return result, e
		}
		intent := a.SigningSecurityIntent{NamespaceID: profile.Binding.NamespaceID, ExpectedNamespaceVersion: identity.NamespaceVersion, Operation: "register_signing_key", EnrollmentID: challenge.ID, Proof: &proof}
		raw, e := signing.Canonical(intent)
		if e != nil {
			return result, e
		}
		id, e := newLocalIDV2()
		if e != nil {
			return result, e
		}
		profile.Local.PendingOperations = []PendingOperationV2{{ID: id, Operation: "register_signing_key", State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: profile.Binding.NamespaceID}, IdempotencyKey: "profile-enrollment-" + id.String(), Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw)}}
	}
	if e = profile.SealBinding(token); e != nil {
		return result, e
	}
	if e = w.CreateV2(profilePathV2(name), profile); e != nil {
		return result, e
	}
	if enrollment != nil {
		return recoverProfileEnrollmentV2(ctx, w, profile, client, token)
	}
	result.Data = compactProfileV2(profile)
	return result, nil
}
func recoverProfileEnrollmentV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte) (Output, error) {
	result := Output{Operation: "profile recover", Data: compactProfileV2(profile)}
	if _, e := checkProfileIdentityV2(ctx, client, profile, nil); e != nil {
		return result, e
	}
	for len(profile.Local.PendingOperations) > 0 {
		pending := profile.Local.PendingOperations[0]
		if pending.Operation != "register_signing_key" {
			return result, fmt.Errorf("non-enrollment pending operation preserved for work recover")
		}
		raw, e := base64.StdEncoding.Strict().DecodeString(pending.Payload)
		if e != nil || signing.Digest(raw) != pending.PayloadDigest {
			return result, fmt.Errorf("pending intent digest differs")
		}
		var intent a.SigningSecurityIntent
		if e = signing.DecodeStrict(raw, &intent, signing.MaxPayloadBytes); e != nil {
			return result, e
		}
		if intent.Operation != pending.Operation || intent.NamespaceID != profile.Binding.NamespaceID || intent.Proof == nil {
			return result, fmt.Errorf("pending registration binding differs")
		}
		before, e := w.ReadV2(profilePathV2(profile.Name))
		if e != nil {
			return result, e
		}
		profile.Local.PendingOperations[0].State = "sent_unknown"
		if e = profile.SealBinding(token); e != nil {
			return result, e
		}
		if e = w.WriteV2(profilePathV2(profile.Name), profile, signing.Digest(before)); e != nil {
			return result, e
		}
		accepted, e := client.SigningMutation(ctx, pending.IdempotencyKey, intent)
		if e != nil {
			result.RequiresAction = "profile recover replays only the frozen enrollment intention"
			return result, e
		}
		result.Committed = true
		if accepted.Key == nil || accepted.Key.ID != profile.Signing.KeyID || accepted.Key.Fingerprint != profile.Signing.PublicKeyFingerprint || accepted.Key.PrincipalID != profile.Binding.PrincipalID {
			return result, fmt.Errorf("registration response differs; pending intent preserved")
		}
		current, e := w.ReadV2(profilePathV2(profile.Name))
		if e != nil {
			return result, e
		}
		// Compare with the last bytes written rather than trusting a fresh editor's
		// content as our expected version after the network call.
		expected, e := EncodeV2Document(profile)
		if e != nil {
			return result, e
		}
		if signing.Digest(current) != signing.Digest(expected) {
			return result, fmt.Errorf("local_version_conflict: registration accepted, edited profile preserved")
		}
		profile.Local.PendingOperations = profile.Local.PendingOperations[1:]
		if e = profile.SealBinding(token); e != nil {
			return result, e
		}
		if e = w.WriteV2(profilePathV2(profile.Name), profile, signing.Digest(expected)); e != nil {
			return result, e
		}
	}
	result.Data = compactProfileV2(profile)
	return result, nil
}
