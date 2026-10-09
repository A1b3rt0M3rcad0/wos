package woscli

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// These are explicitly unsigned legacy records. They never implement the signed
// contract discriminator or carry a fabricated issuer/agent proof.
type LegacyOriginalV2 struct {
	Path   string   `json:"path"`
	Digest string   `json:"digest"`
	Chunks []string `json:"chunks"`
}
type LegacyUnsignedFileV2 struct {
	SchemaVersion     int                `json:"schema_version"`
	Kind              string             `json:"kind"`
	Protocol          string             `json:"protocol"`
	SourceWorkspace   string             `json:"source_workspace"`
	SourceOrigin      string             `json:"source_origin"`
	SourceNamespaceID d.ID               `json:"source_namespace_id"`
	Contract          ContractDocument   `json:"contract"`
	Checkpoint        *CheckpointDraft   `json:"checkpoint,omitempty"`
	Result            *ResultDraft       `json:"result,omitempty"`
	Originals         []LegacyOriginalV2 `json:"originals"`
}
type MigrationTargetV2 struct {
	ContractID d.ID   `json:"contract_id"`
	Digest     string `json:"digest"`
}
type FrozenMigrationV2 struct {
	Source          string              `json:"source"`
	InventoryDigest string              `json:"inventory_digest"`
	Targets         []MigrationTargetV2 `json:"targets"`
}

func convertedLegacyFilesV2(source *Workspace, config Config, inventory MigrationInventoryV2) ([]LegacyUnsignedFileV2, error) {
	files := []LegacyUnsignedFileV2{}
	seen := map[d.ID]bool{}
	for _, entry := range inventory.Files {
		if entry.Category != "contract" {
			continue
		}
		var contract ContractDocument
		raw, e := source.ReadV2(filepath.FromSlash(entry.Path))
		if e != nil {
			return nil, e
		}
		if e = DecodeDocument(raw, &contract); e != nil {
			return nil, e
		}
		c := contract.Contract
		if contract.SchemaVersion != 1 || contract.Kind != "WorkContract" || c.Validate() != nil || c.Scope.NamespaceID != config.Scope.NamespaceID || c.SignedBinding != nil || seen[c.ID] {
			return nil, fmt.Errorf("legacy contract is invalid, signed or ambiguous; preserve originals")
		}
		dir := filepath.Dir(filepath.FromSlash(entry.Path))
		if dir != stateDir(c.WorkItemID, c.ID) {
			return nil, fmt.Errorf("legacy contract path/identity differs")
		}
		seen[c.ID] = true
		file := LegacyUnsignedFileV2{SchemaVersion: 2, Kind: "WOSLegacyUnsignedContract", Protocol: "legacy_unsigned", SourceOrigin: config.Connection.ServerURL, SourceWorkspace: source.canonicalPath, SourceNamespaceID: config.Scope.NamespaceID, Contract: contract, Originals: []LegacyOriginalV2{}}
		for _, original := range inventory.Files {
			path := filepath.FromSlash(original.Path)
			if !strings.HasPrefix(path, dir+string(filepath.Separator)) {
				continue
			}
			// Custom files remain untouched in the original workspace. Only known
			// operational records are included, with their exact bytes and digests.
			if original.Category == "other" && filepath.Base(path) != "state.json" {
				continue
			}
			content, e := source.ReadV2(path)
			if e != nil {
				return nil, e
			}
			if signing.Digest(content) != original.Digest {
				return nil, fmt.Errorf("source changed during conversion")
			}
			observed := content
			encoded := LegacyOriginalV2{Path: original.Path, Digest: original.Digest, Chunks: []string{}}
			for len(content) > 0 {
				n := len(content)
				if n > 48<<10 {
					n = 48 << 10
				}
				encoded.Chunks = append(encoded.Chunks, base64.StdEncoding.EncodeToString(content[:n]))
				content = content[n:]
			}
			file.Originals = append(file.Originals, encoded)
			switch original.Category {
			case "checkpoint_draft":
				var draft CheckpointDraft
				if e = DecodeDocument(observed, &draft); e != nil {
					return nil, e
				}
				if draft.SchemaVersion != 1 || draft.Kind != "WorkCheckpointDraft" || draft.ContractID != c.ID || draft.SpecDigest != c.SpecDigest {
					return nil, fmt.Errorf("legacy checkpoint binding differs")
				}
				file.Checkpoint = &draft
			case "result_draft":
				var draft ResultDraft
				if e = DecodeDocument(observed, &draft); e != nil {
					return nil, e
				}
				if draft.SchemaVersion != 1 || draft.Kind != "WorkResult" || draft.ContractID != c.ID || draft.SpecDigest != c.SpecDigest {
					return nil, fmt.Errorf("legacy result binding differs")
				}
				file.Result = &draft
			}
		}
		if _, e = EncodeV2Document(file); e != nil {
			return nil, e
		}
		files = append(files, file)
		if len(files) > 100 {
			return nil, fmt.Errorf("migration exceeds local contract limit")
		}
	}
	return files, nil
}

func convertWorkspaceV2(ctx context.Context, source *Workspace, config Config, inventory MigrationInventoryV2, o options) (Output, error) {
	result := Output{Operation: "workspace migrate", Data: inventory}
	if len(inventory.UnresolvedIntentions) > 0 || len(inventory.UnmaterializedAcquisitions) > 0 {
		return result, fmt.Errorf("reconcile original prepared/sent_unknown journals before conversion; migration never acquires replacement work")
	}
	if o.values["destination"] == "" || !validProfileName(o.values["profile"]) {
		return result, usage("conversion requires --destination <approved schema-2 workspace> and --profile <name>")
	}
	destination, e := OpenWorkspace(o.values["destination"])
	if e != nil {
		return result, e
	}
	defer destination.Close()
	if destination.canonicalPath == source.canonicalPath || strings.HasPrefix(destination.canonicalPath, filepath.Join(source.canonicalPath, ".wos")+string(filepath.Separator)) {
		return result, fmt.Errorf("migration destination must be separate from original operational files")
	}
	profile, client, token, e := loadProfileClientV2(destination, o.values["profile"], defaultSecretResolverV2(destination))
	if e != nil {
		return result, e
	}
	defer clear(token)
	targetIdentity, e := checkProfileIdentityV2(ctx, client, profile, nil)
	if e != nil {
		return result, e
	}
	if profile.Binding.ServerOrigin != config.Connection.ServerURL || profile.Binding.NamespaceID != config.Scope.NamespaceID {
		return result, fmt.Errorf("approved destination differs from legacy workspace")
	}
	_, sourceClient, e := loadClient(source)
	if e != nil {
		return result, e
	}
	sourceIdentity, e := sourceClient.SigningIdentity(ctx, nil)
	if e != nil {
		return result, e
	}
	if sourceIdentity.ServerID != targetIdentity.ServerID || sourceIdentity.CredentialID != targetIdentity.CredentialID || sourceIdentity.PrincipalID != targetIdentity.PrincipalID {
		return result, fmt.Errorf("migration requires the original approved credential identity")
	}
	files, e := convertedLegacyFilesV2(source, config, inventory)
	if e != nil {
		return result, e
	}
	manifest := FrozenMigrationV2{Source: source.canonicalPath, InventoryDigest: inventory.InventoryDigest, Targets: []MigrationTargetV2{}}
	for _, file := range files {
		c := file.Contract.Contract
		live, e := client.GetWorkContract(ctx, c.Scope, c.ID)
		if e != nil {
			return result, e
		}
		if live.Value.Contract.ID != c.ID || live.Value.Contract.Scope != c.Scope || live.Value.Contract.WorkItemID != c.WorkItemID || live.Value.Contract.SpecDigest != c.SpecDigest || live.Value.Contract.HolderPrincipalID != c.HolderPrincipalID {
			return result, fmt.Errorf("remote original contract binding differs")
		}
		raw, e := EncodeV2Document(file)
		if e != nil {
			return result, e
		}
		manifest.Targets = append(manifest.Targets, MigrationTargetV2{c.ID, signing.Digest(raw)})
	}
	frozen, e := signing.Canonical(manifest)
	if e != nil || len(frozen) > 180<<10 {
		return result, fmt.Errorf("migration manifest exceeds bound")
	}
	var intent PendingOperationV2
	alreadyVerified := false
	// Short profile locks protect local state; all network preflight is above.
	e = mutateProfileV2(ctx, destination, profile, token, func(current *ProfileV2) error {
		for _, pending := range current.Local.PendingOperations {
			if pending.Operation != "WorkspaceMigration" {
				return fmt.Errorf("reconcile destination operations before migration")
			}
			decoded, e := base64.StdEncoding.Strict().DecodeString(pending.Payload)
			if e != nil || string(decoded) != string(frozen) {
				return fmt.Errorf("original migration source/inventory differs; preserve partial conversion")
			}
			intent = pending
		}
		if intent.ID.IsZero() {
			ids, e := contractIDsV2(destination, current.Name)
			if e != nil {
				return e
			}
			if len(ids) != 0 {
				if len(ids) != len(manifest.Targets) {
					return fmt.Errorf("new migration requires an empty approved contract directory")
				}
				for _, target := range manifest.Targets {
					release, e := destination.LockV2(ctx, profile.Name, target.ContractID.String())
					if e != nil {
						return e
					}
					raw, e := destination.ReadV2(contractPathV2(profile.Name, target.ContractID))
					release()
					if e != nil {
						return e
					}
					if signing.Digest(raw) != target.Digest {
						return fmt.Errorf("existing converted draft differs; preserve it")
					}
				}
				alreadyVerified = true
				return nil
			}
			id, e := newLocalIDV2()
			if e != nil {
				return e
			}
			key, e := sdk.NewIdempotencyKey()
			if e != nil {
				return e
			}
			intent = PendingOperationV2{ID: id, Operation: "WorkspaceMigration", State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: profile.Binding.NamespaceID}, IdempotencyKey: key, Payload: base64.StdEncoding.EncodeToString(frozen), PayloadDigest: signing.Digest(frozen)}
			current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
		}
		return nil
	})
	if e != nil {
		return result, e
	}
	if alreadyVerified {
		current, _, e := migrationInventoryV2(source)
		if e != nil {
			return result, e
		}
		if current.InventoryDigest != manifest.InventoryDigest {
			return result, fmt.Errorf("source changed while verifying existing conversion")
		}
		result.Data = map[string]any{"inventory": inventory, "converted_contracts": len(files), "originals_preserved": true, "already_verified": true, "protocol": "legacy_unsigned"}
		return result, nil
	}
	e = publishMigrationV2(ctx, source, destination, profile, token, intent, manifest, files, nil)
	result.Data = map[string]any{"inventory": inventory, "converted_contracts": len(files), "originals_preserved": true, "protocol": "legacy_unsigned"}
	if e != nil {
		result.RequiresAction = "repeat migration with the same source, destination and profile; edited bytes are preserved"
		return result, e
	}
	result.RequiresAction = "review converted drafts and retained source records before explicitly discarding any originals; unsigned records cannot authorize signed work"
	return result, nil
}
func publishMigrationV2(ctx context.Context, source, destination *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, manifest FrozenMigrationV2, files []LegacyUnsignedFileV2, hook func(int) error) error {
	for i, file := range files {
		e := mutateProfileV2(ctx, destination, profile, token, func(current *ProfileV2) error {
			if _, e := pendingIndexV2(current, intent); e != nil {
				return e
			}
			release, e := destination.LockV2(ctx, profile.Name, file.Contract.Contract.ID.String())
			if e != nil {
				return e
			}
			defer release()
			path := contractPathV2(profile.Name, file.Contract.Contract.ID)
			expected, e := EncodeV2Document(file)
			if e != nil {
				return e
			}
			existing, e := destination.ReadV2(path)
			if os.IsNotExist(e) {
				return destination.CreateV2(path, file)
			}
			if e != nil {
				return e
			}
			if signing.Digest(existing) != signing.Digest(expected) {
				return fmt.Errorf("edited converted contract preserved")
			}
			return nil
		})
		if e != nil {
			return e
		}
		if hook != nil {
			if e = hook(i); e != nil {
				return e
			}
		}
	}
	current, _, e := migrationInventoryV2(source)
	if e != nil {
		return e
	}
	if current.InventoryDigest != manifest.InventoryDigest {
		return fmt.Errorf("source changed; partial migration and original drafts preserved")
	}
	return mutateProfileV2(ctx, destination, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		for _, target := range manifest.Targets {
			release, e := destination.LockV2(ctx, profile.Name, target.ContractID.String())
			if e != nil {
				return e
			}
			raw, e := destination.ReadV2(contractPathV2(profile.Name, target.ContractID))
			release()
			if e != nil {
				return e
			}
			if signing.Digest(raw) != target.Digest {
				return fmt.Errorf("converted draft changed; migration intention preserved")
			}
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
}
