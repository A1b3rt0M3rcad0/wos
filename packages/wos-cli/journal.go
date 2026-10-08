package woscli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"path/filepath"
	"strings"
	"time"
)

type Destination struct {
	ServerURL     string    `json:"server_url"`
	CredentialRef string    `json:"credential_ref"`
	NamespaceID   domain.ID `json:"namespace_id"`
}

func (c Config) Destination() Destination {
	return Destination{strings.TrimRight(c.Connection.ServerURL, "/"), c.Connection.CredentialRef, c.Scope.NamespaceID}
}

type FrozenPayload string

func (p FrozenPayload) MarshalJSON() ([]byte, error) { return json.Marshal(string(p)) }
func (p *FrozenPayload) UnmarshalJSON(raw []byte) error {
	var value string
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		*p = FrozenPayload(value)
		return nil
	}
	if !json.Valid(raw) {
		return fmt.Errorf("invalid frozen payload")
	}
	*p = FrozenPayload(raw)
	return nil
}

type Intent struct {
	SchemaVersion  int           `json:"schema_version"`
	Destination    Destination   `json:"destination"`
	Scope          domain.Scope  `json:"scope"`
	Command        string        `json:"command"`
	IdempotencyKey string        `json:"idempotency_key"`
	Payload        FrozenPayload `json:"payload"`
	PayloadDigest  string        `json:"payload_digest"`
	State          string        `json:"state"`
	PreparedAt     time.Time     `json:"prepared_at"`
}
type Receipt struct {
	SchemaVersion  int             `json:"schema_version"`
	Destination    Destination     `json:"destination"`
	IdempotencyKey string          `json:"idempotency_key"`
	PayloadDigest  string          `json:"payload_digest"`
	Response       json.RawMessage `json:"response"`
}
type LocalError struct {
	Err         error
	Committed   bool
	ReceiptPath string
}

func (e *LocalError) Error() string { return "local journal/materialization: " + e.Err.Error() }
func (e *LocalError) Unwrap() error { return e.Err }
func (w *Workspace) saveJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return w.AtomicWrite(path, raw)
}
func (w *Workspace) Prepare(config Config, scope domain.Scope, dir, name string, command any) (string, Intent, error) {
	payload, err := commands.Encode(command)
	if raw, ok := command.(json.RawMessage); ok {
		payload = append([]byte(nil), raw...)
		err = nil
		if !json.Valid(payload) {
			err = fmt.Errorf("invalid frozen command")
		}
	}
	if err != nil {
		return "", Intent{}, err
	}
	digest, err := domain.SemanticDigest(json.RawMessage(payload))
	if err != nil {
		return "", Intent{}, err
	}
	key, err := sdk.NewIdempotencyKey()
	if err != nil {
		return "", Intent{}, err
	}
	intent := Intent{SchemaVersion: 1, Destination: config.Destination(), Scope: scope, Command: name, IdempotencyKey: key, Payload: FrozenPayload(payload), PayloadDigest: digest, State: "prepared", PreparedAt: time.Now().UTC()}
	path := filepath.Join(dir, "outbox", key+".json")
	if err = w.saveJSON(path, intent); err != nil {
		return "", Intent{}, &LocalError{Err: err}
	}
	return path, intent, nil
}
func (w *Workspace) Replay(ctx context.Context, client *sdk.Client, config Config, path string) (json.RawMessage, string, error) {
	raw, err := w.Read(path)
	if err != nil {
		return nil, "", &LocalError{Err: err}
	}
	var intent Intent
	if err = json.Unmarshal(raw, &intent); err != nil {
		return nil, "", &LocalError{Err: err}
	}
	if intent.SchemaVersion != 1 || intent.Destination != config.Destination() || intent.Scope.NamespaceID != config.Scope.NamespaceID {
		return nil, "", fmt.Errorf("journal destination differs from trusted workspace")
	}
	digest, err := domain.SemanticDigest(json.RawMessage(intent.Payload))
	if err != nil || digest != intent.PayloadDigest {
		return nil, "", fmt.Errorf("journal payload digest differs")
	}
	receiptPath := filepath.Join(filepath.Dir(filepath.Dir(path)), "receipts", intent.IdempotencyKey+".json")
	if intent.State == "confirmed" {
		data, err := w.Read(receiptPath)
		if err != nil {
			return nil, receiptPath, &LocalError{Err: err, Committed: true, ReceiptPath: receiptPath}
		}
		var receipt Receipt
		if err = json.Unmarshal(data, &receipt); err != nil {
			return nil, receiptPath, &LocalError{Err: err, Committed: true}
		}
		if receipt.Destination != intent.Destination || receipt.PayloadDigest != intent.PayloadDigest || receipt.IdempotencyKey != intent.IdempotencyKey {
			return nil, receiptPath, fmt.Errorf("receipt binding differs")
		}
		return receipt.Response, receiptPath, nil
	}
	if intent.State == "rejected" {
		return nil, receiptPath, fmt.Errorf("intent was rejected; inspect before creating a new intent")
	}
	if intent.State != "prepared" && intent.State != "sent_unknown" {
		return nil, receiptPath, fmt.Errorf("unknown journal state")
	}
	intent.State = "sent_unknown"
	if err = w.saveJSON(path, intent); err != nil {
		return nil, receiptPath, &LocalError{Err: err}
	}
	response, err := client.ExecuteCommand(ctx, intent.Command, intent.IdempotencyKey, json.RawMessage(intent.Payload))
	if err != nil {
		var remote *sdk.Error
		if errors.As(err, &remote) && remote.Status < 500 && remote.Status != 408 && remote.Status != 429 {
			intent.State = "rejected"
			if writeErr := w.saveJSON(path, intent); writeErr != nil {
				return nil, receiptPath, &LocalError{Err: writeErr}
			}
		}
		return nil, receiptPath, err
	}
	receipt := Receipt{SchemaVersion: 1, Destination: intent.Destination, IdempotencyKey: intent.IdempotencyKey, PayloadDigest: intent.PayloadDigest, Response: response}
	if err = w.saveJSON(receiptPath, receipt); err != nil {
		return response, receiptPath, &LocalError{Err: err, Committed: true, ReceiptPath: receiptPath}
	}
	intent.State = "confirmed"
	if err = w.saveJSON(path, intent); err != nil {
		return response, receiptPath, &LocalError{Err: err, Committed: true, ReceiptPath: receiptPath}
	}
	return response, receiptPath, nil
}
func (w *Workspace) Mutate(ctx context.Context, client *sdk.Client, c Config, scope domain.Scope, dir, name string, cmd any) (json.RawMessage, string, error) {
	path, _, err := w.Prepare(c, scope, dir, name, cmd)
	if err != nil {
		return nil, "", err
	}
	return w.Replay(ctx, client, c, path)
}
