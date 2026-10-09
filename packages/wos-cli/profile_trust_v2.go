package woscli

import (
	"context"
	"errors"
	"fmt"
	"slices"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Only a pin is appended. Origin, persistent server, CID, Principal, own signing
// identity and all frozen payloads remain unchanged. The old binding MAC is
// authenticated as bounded local trust lineage in the same atomic profile write.
func approveCurrentIssuerV2(ctx context.Context, w *Workspace, name string, o options, resolver SecretResolver) (Output, error) {
	result := Output{Operation: "profile trust"}
	fingerprint := o.values["issuer-fingerprint"]
	if !d.ValidSignedDigest(fingerprint) {
		return result, usage("profile trust requires an independently approved --issuer-fingerprint")
	}
	profile, client, token, e := loadProfileClientV2(w, name, resolver)
	if e != nil {
		return result, e
	}
	defer clear(token)
	// Do not call checkProfileIdentityV2: the explicitly approved new issuer is
	// intentionally outside the old pins. Verify the fixed credential identity.
	identity, e := client.SigningIdentity(ctx, nil)
	if e != nil {
		return result, e
	}
	if identity.ServerID != profile.Binding.ServerID.String() || identity.NamespaceID != profile.Binding.NamespaceID || identity.PrincipalID != profile.Binding.PrincipalID || identity.CredentialID != profile.Binding.CredentialID {
		return result, fmt.Errorf("remote identity differs from approved profile")
	}
	trust, e := client.GetSignedTrust(ctx, profile.Binding.NamespaceID)
	if e != nil {
		return result, e
	}
	if trust.Server == nil || trust.Server.Validate() != nil || trust.Server.ID != profile.Binding.ServerID || trust.Server.Fingerprint != fingerprint {
		return result, fmt.Errorf("current issuer differs from explicitly approved server/fingerprint")
	}
	pin := ProfileIssuerV2{KeyID: trust.Server.IssuerKeyID, PublicKey: trust.Server.PublicKey, Fingerprint: fingerprint}
	changed := false
	alreadyApproved := errors.New("issuer already explicitly approved")
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		for _, previous := range current.Binding.IssuerKeys {
			if previous.KeyID == pin.KeyID || previous.Fingerprint == pin.Fingerprint {
				if previous != pin {
					return fmt.Errorf("issuer identity/fingerprint is already bound differently")
				}
				return alreadyApproved
			}
		}
		if len(current.Binding.IssuerKeys) >= 10 || len(current.Local.PriorIssuerBindings) >= 9 {
			return fmt.Errorf("approved issuer trust bound reached; retain original profiles and pins")
		}
		digest, e := current.issuerIndependentIdentityDigest()
		if e != nil {
			return e
		}
		// Copies avoid aliasing the caller's currently verified immutable pin set.
		current.Binding.IssuerKeys = append(slices.Clone(current.Binding.IssuerKeys), pin)
		current.Local.PriorIssuerBindings = append(slices.Clone(current.Local.PriorIssuerBindings), PriorIssuerBindingV2{BindingMAC: current.Local.BindingMAC, IdentityDigest: digest})
		changed = true
		return current.Validate()
	})
	if errors.Is(e, alreadyApproved) {
		e = nil
	}
	if e != nil {
		result.RequiresAction = "repeat explicit trust approval with the same fingerprint; pending original intentions remain intact"
		return result, e
	}
	result.Data = map[string]any{"profile": name, "issuer_key_id": pin.KeyID, "issuer_fingerprint": pin.Fingerprint, "appended": changed, "historical_pins_retained": true, "frozen_returns_unchanged": true}
	return result, nil
}
