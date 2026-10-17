package woscli

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

// projectTenantProjection accepts the previous local project address for the
// explicit migration command and for reading an unmodified shared project file.
// It never applies to incoming commands, pending payloads or signed documents.
func projectTenantProjection(raw []byte) ([]byte, bool, error) {
	data, err := YAMLJSONV2(raw)
	if err != nil {
		return nil, false, err
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(data, &doc); err != nil {
		return nil, false, err
	}
	var scope map[string]json.RawMessage
	if err = json.Unmarshal(doc["scope"], &scope); err != nil {
		return nil, false, err
	}
	old, legacy := scope["namespace_id"]
	if !legacy {
		return data, false, nil
	}
	if _, both := scope["tenant_id"]; both {
		return nil, false, fmt.Errorf("project contains conflicting legacy and current Tenant fields")
	}
	delete(scope, "namespace_id")
	scope["tenant_id"] = old
	doc["scope"], err = json.Marshal(scope)
	if err != nil {
		return nil, false, err
	}
	data, err = json.Marshal(doc)
	return data, true, err
}

func legacyProfileProjection(raw []byte) (ProfileV2, []byte, error) {
	var profile ProfileV2
	data, err := YAMLJSONV2(raw)
	if err != nil {
		return profile, nil, err
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(data, &doc); err != nil {
		return profile, nil, err
	}
	var binding map[string]json.RawMessage
	if err = json.Unmarshal(doc["binding"], &binding); err != nil {
		return profile, nil, err
	}
	old, legacy := binding["namespace_id"]
	if !legacy {
		return profile, nil, fmt.Errorf("profile already uses tenant_id; no migration is needed")
	}
	if _, both := binding["tenant_id"]; both {
		return profile, nil, fmt.Errorf("profile contains conflicting legacy and current Tenant fields")
	}
	var local struct {
		PendingOperations []json.RawMessage `json:"pending_operations"`
	}
	if err = json.Unmarshal(doc["_local"], &local); err != nil {
		return profile, nil, err
	}
	if len(local.PendingOperations) != 0 {
		return profile, nil, fmt.Errorf("protocol-2 pending intentions must be reconciled with the original client before migration; preserve all frozen payloads and signatures")
	}
	delete(binding, "namespace_id")
	binding["tenant_id"] = old
	doc["binding"], err = json.Marshal(binding)
	if err != nil {
		return profile, nil, err
	}
	projected, err := json.Marshal(doc)
	if err != nil {
		return profile, nil, err
	}
	if err = DecodeV2Document(projected, &profile); err != nil {
		return profile, nil, err
	}
	// Reconstruct only the old authenticated local profile header. No domain or
	// frozen document passes through this field-name projection.
	message, err := profile.bindingMessage()
	if err != nil {
		return profile, nil, err
	}
	var header map[string]json.RawMessage
	if err = json.Unmarshal(message, &header); err != nil {
		return profile, nil, err
	}
	delete(binding, "tenant_id")
	binding["namespace_id"] = old
	header["binding"], err = json.Marshal(binding)
	if err != nil {
		return profile, nil, err
	}
	oldMessage, err := signing.Canonical(header)
	return profile, oldMessage, err
}

func verifyLegacyProfileMAC(profile ProfileV2, message, token []byte) error {
	if len(token) < 32 {
		return fmt.Errorf("credential unavailable")
	}
	expected, err := hex.DecodeString(profile.Local.BindingMAC)
	if err != nil {
		return fmt.Errorf("legacy profile binding proof is invalid")
	}
	mac := hmac.New(sha256.New, token)
	mac.Write([]byte("WOS local profile destination v2\x00"))
	mac.Write(message)
	if !hmac.Equal(expected, mac.Sum(nil)) {
		return fmt.Errorf("legacy profile destination/identity proof differs; original bytes were preserved")
	}
	pending, err := json.Marshal(struct {
		Name                string                 `json:"name"`
		BindingMAC          string                 `json:"binding_mac"`
		Operations          []PendingOperationV2   `json:"operations"`
		PriorIssuerBindings []PriorIssuerBindingV2 `json:"prior_issuer_bindings,omitempty"`
	}{profile.Name, profile.Local.BindingMAC, profile.Local.PendingOperations, profile.Local.PriorIssuerBindings})
	if err != nil {
		return err
	}
	mac = hmac.New(sha256.New, token)
	mac.Write([]byte("WOS local profile pending v2\x00"))
	mac.Write(pending)
	expected, err = hex.DecodeString(profile.Local.PendingMAC)
	if err != nil || !hmac.Equal(expected, mac.Sum(nil)) {
		return fmt.Errorf("legacy profile pending proof differs; original bytes were preserved")
	}
	return nil
}

func migrateTenantProfile(ctx context.Context, w *Workspace, name string, resolver SecretResolver) (Output, error) {
	result := Output{Operation: "profile migrate-tenant"}
	release, err := w.LockV2(ctx, name, "")
	if err != nil {
		return result, err
	}
	defer release()
	profilePath, err := w.DocumentPath(filepath.Join(".wos/profiles", name, "profile"))
	if err != nil {
		return result, err
	}
	original, err := w.ReadV2(profilePath)
	if err != nil {
		return result, err
	}
	profile, message, err := legacyProfileProjection(original)
	if err != nil {
		return result, err
	}
	if profile.Name != name {
		return result, fmt.Errorf("legacy profile directory/header differ")
	}
	token, err := resolver.Read(profile.Authentication.CredentialRef)
	if err != nil {
		return result, err
	}
	defer clear(token)
	if err = verifyLegacyProfileMAC(profile, message, token); err != nil {
		return result, err
	}
	// Prior bindings authenticate pending returns. With no pending operations,
	// retain them in the original backup and establish a new current local MAC.
	profile.Local.PriorIssuerBindings = nil
	if err = profile.Validate(); err != nil {
		return result, err
	}
	project, err := w.LoadProjectV2()
	if err != nil {
		return result, err
	}
	if project.Scope.TenantID != profile.Binding.TenantID || project.Connection.ServerURL != profile.Binding.ServerOrigin || project.Connection.ExpectedServerID != profile.Binding.ServerID {
		return result, fmt.Errorf("legacy project/profile destination differs")
	}
	ids, err := contractIDsV2(w, name)
	if err != nil {
		return result, err
	}
	for _, id := range ids {
		_, legacy, err := readLegacyUnsignedV2(w, profile, id)
		if err != nil {
			return result, err
		}
		if legacy {
			continue
		}
		file, _, _, _, err := loadContractV2(w, profile, id)
		if err != nil {
			return result, err
		}
		if file.Local.Pending != nil {
			return result, fmt.Errorf("contract %s has frozen pending intent; reconcile it with the original client before migration", id)
		}
	}
	if err = profile.SealBinding(token); err != nil {
		return result, err
	}
	backup := filepath.Join(".wos/tenant-migration", name, "profile.original")
	if saved, readErr := w.ReadV2(backup); readErr == nil {
		if signing.Digest(saved) != signing.Digest(original) {
			return result, fmt.Errorf("existing migration backup differs; original files were preserved")
		}
	} else if !os.IsNotExist(readErr) {
		return result, readErr
	} else if err = w.writeRawV2(backup, original, "", true); err != nil {
		return result, err
	}
	// Each replacement is atomic and digest-guarded. Project projection keeps a
	// crash between these writes readable, and the original profile is retained.
	if err = w.WriteV2(profilePath, profile, signing.Digest(original)); err != nil {
		return result, err
	}
	projectPath, err := w.DocumentPath(".wos/project")
	if err != nil {
		return result, err
	}
	projectRaw, err := w.ReadV2(projectPath)
	if err != nil {
		return result, err
	}
	_, legacy, err := projectTenantProjection(projectRaw)
	if err != nil {
		return result, err
	}
	if legacy {
		backupProject := filepath.Join(".wos/tenant-migration", name, "project.original")
		if _, err = w.ReadV2(backupProject); os.IsNotExist(err) {
			err = w.writeRawV2(backupProject, projectRaw, "", true)
		}
		if err != nil {
			return result, err
		}
		if err = w.WriteV2(projectPath, project, signing.Digest(projectRaw)); err != nil {
			return result, err
		}
	}
	result.Data = map[string]any{"profile": compactProfileV2(profile), "original_profile": backup, "historical_contracts": len(ids), "pending_operations": 0}
	return result, nil
}
