package woscli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

type MigrationFileV2 struct {
	Path     string `json:"path"`
	Digest   string `json:"digest"`
	Bytes    int    `json:"bytes"`
	Category string `json:"category"`
}
type MigrationInventoryV2 struct {
	SourceSchema               int               `json:"source_schema"`
	TargetSchema               int               `json:"target_schema"`
	Files                      []MigrationFileV2 `json:"files"`
	InventoryDigest            string            `json:"inventory_digest"`
	UnmaterializedAcquisitions []string          `json:"unmaterialized_acquisitions"`
	UnresolvedIntentions       []string          `json:"unresolved_intentions"`
	TotalBytes                 int               `json:"total_bytes"`
	DryRun                     bool              `json:"dry_run"`
	OriginalsPreserved         bool              `json:"originals_preserved"`
}

// The inventory performs no network calls and never invokes journal Replay.
// Its digest covers exact bytes, including drafts and original receipt bodies.
func migrationInventoryV2(w *Workspace) (MigrationInventoryV2, Config, error) {
	r := MigrationInventoryV2{SourceSchema: 1, TargetSchema: 2, Files: []MigrationFileV2{}, UnmaterializedAcquisitions: []string{}, UnresolvedIntentions: []string{}, DryRun: true, OriginalsPreserved: true}
	var config Config
	path, e := w.DocumentPath(".wos/config")
	if e != nil {
		return r, config, e
	}
	if e = w.ReadDocument(path, &config); e != nil {
		return r, config, e
	}
	if e = config.Validate(); e != nil {
		return r, config, e
	}
	raw, e := w.ReadV2(".wos/trust.json")
	if e != nil {
		return r, config, e
	}
	var destination Destination
	if e = json.Unmarshal(raw, &destination); e != nil || destination != config.Destination() {
		return r, config, fmt.Errorf("legacy destination differs from approved binding")
	}
	entries := 0
	var visit func(string, int) error
	visit = func(path string, depth int) error {
		if depth > 12 {
			return fmt.Errorf("legacy inventory depth bound exceeded")
		}
		if e := w.check(path); e != nil {
			return e
		}
		directory, e := w.root.Open(path)
		if e != nil {
			return e
		}
		children, e := directory.ReadDir(513)
		directory.Close()
		if e != nil && e != io.EOF {
			return e
		}
		if len(children) > 512 {
			return fmt.Errorf("legacy inventory directory bound exceeded")
		}
		sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
		for _, child := range children {
			entries++
			if entries > 512 {
				return fmt.Errorf("legacy inventory entry bound exceeded")
			}
			next := filepath.Join(path, child.Name())
			if child.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("legacy symlink preserved; explicit reconciliation required")
			}
			if child.IsDir() {
				if e = visit(next, depth+1); e != nil {
					return e
				}
				continue
			}
			content, e := w.ReadV2(next)
			if e != nil {
				return e
			}
			r.TotalBytes += len(content)
			if r.TotalBytes > 32<<20 {
				return fmt.Errorf("legacy inventory byte bound exceeded")
			}
			category := "other"
			switch stem := strings.TrimSuffix(child.Name(), filepath.Ext(child.Name())); stem {
			case "contract":
				category = "contract"
			case "checkpoint":
				category = "checkpoint_draft"
			case "result":
				category = "result_draft"
			}
			if filepath.Base(filepath.Dir(next)) == "outbox" && filepath.Ext(next) == ".json" {
				category = "intention"
				var intent Intent
				if e = json.Unmarshal(content, &intent); e != nil {
					return fmt.Errorf("invalid legacy intention")
				}
				digest, e := d.SemanticDigest(json.RawMessage(intent.Payload))
				if e != nil || digest != intent.PayloadDigest || intent.Destination != config.Destination() || intent.Scope.NamespaceID != config.Scope.NamespaceID || intent.SchemaVersion != 1 || d.ValidateIdempotencyKey(intent.IdempotencyKey) != nil {
					return fmt.Errorf("legacy intention binding/digest differs")
				}
				switch intent.State {
				case "prepared", "sent_unknown":
					r.UnresolvedIntentions = append(r.UnresolvedIntentions, filepath.ToSlash(next))
				case "confirmed":
					receiptPath := filepath.Join(filepath.Dir(filepath.Dir(next)), "receipts", intent.IdempotencyKey+".json")
					receiptRaw, e := w.ReadV2(receiptPath)
					if e != nil {
						return e
					}
					var receipt Receipt
					if e = json.Unmarshal(receiptRaw, &receipt); e != nil || receipt.SchemaVersion != 1 || receipt.Destination != intent.Destination || receipt.IdempotencyKey != intent.IdempotencyKey || receipt.PayloadDigest != intent.PayloadDigest || !json.Valid(receipt.Response) {
						return fmt.Errorf("legacy confirmed receipt differs")
					}
					if intent.Command == "acquire_work_contract" || intent.Command == "acquire_next_work_contract" {
						var acquired sdk.CommandResult[a.WorkContractResult]
						decodeErr := json.Unmarshal(receipt.Response, &acquired)
						contract := acquired.Value.Contract
						empty := false
						if intent.Command == "acquire_next_work_contract" && contract.ID.IsZero() {
							var next sdk.CommandResult[a.WorkContractAcquisition]
							decodeErr = json.Unmarshal(receipt.Response, &next)
							if next.Value.Result != nil {
								contract = next.Value.Result.Contract
							}
							empty = !next.ResultOmitted && !next.Value.Acquired && !next.CommandID.IsZero()
						}
						if !empty {
							missing := decodeErr != nil || acquired.ResultOmitted || contract.Validate() != nil || contract.Scope != intent.Scope
							if !missing {
								path, e := w.DocumentPath(filepath.Join(stateDir(contract.WorkItemID, contract.ID), "contract"))
								if e != nil {
									return e
								}
								_, e = w.ReadV2(path)
								missing = os.IsNotExist(e)
								if e != nil && !missing {
									return e
								}
							}
							if missing {
								r.UnmaterializedAcquisitions = append(r.UnmaterializedAcquisitions, filepath.ToSlash(next))
							}
						}
					}
				case "rejected":
				default:
					return fmt.Errorf("unknown legacy intention state preserved")
				}
			} else if filepath.Base(filepath.Dir(next)) == "receipts" && filepath.Ext(next) == ".json" {
				category = "receipt"
			}
			r.Files = append(r.Files, MigrationFileV2{filepath.ToSlash(next), signing.Digest(content), len(content), category})
		}
		return nil
	}
	if e = visit(".wos", 0); e != nil {
		return r, config, e
	}
	bytes, e := signing.Canonical(r.Files)
	if e != nil {
		return r, config, e
	}
	r.InventoryDigest = signing.Digest(bytes)
	return r, config, nil
}
func workspaceMigrationCommandV2(ctx context.Context, w *Workspace, o options) (Output, error) {
	result := Output{Operation: "workspace migrate"}
	if len(o.args) != 2 || o.args[1] != "migrate" || o.values["to"] != "2" {
		return result, usage("workspace migrate --to 2 --profile <name> --dry-run")
	}
	if o.values["profile"] != "" && !validProfileName(o.values["profile"]) {
		return result, usage("invalid destination profile name")
	}
	inventory, config, e := migrationInventoryV2(w)
	if e != nil {
		return result, e
	}
	result.Data = inventory
	if o.values["dry-run"] != "true" {
		inventory, config, e = func() (MigrationInventoryV2, Config, error) {
			release, e := w.LockV2(ctx, "workspace_migration", "")
			if e != nil {
				return MigrationInventoryV2{}, Config{}, e
			}
			defer release()
			return migrationInventoryV2(w)
		}()
		if e != nil {
			return result, e
		}
		return convertWorkspaceV2(ctx, w, config, inventory, o)
	}
	if len(inventory.UnresolvedIntentions) > 0 || len(inventory.UnmaterializedAcquisitions) > 0 {
		result.RequiresAction = "reconcile the original legacy intentions before conversion; dry-run never sends them"
	}
	return result, nil
}
