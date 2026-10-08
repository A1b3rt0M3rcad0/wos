package woscli

import (
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/url"
	"regexp"
)

type Config struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	Profile       string           `json:"profile"`
	Connection    ConnectionConfig `json:"connection"`
	Scope         WorkspaceScope   `json:"scope"`
	Workspace     WorkspaceConfig  `json:"workspace"`
	Lease         LeaseConfig      `json:"lease"`
	Output        OutputConfig     `json:"output"`
}
type ConnectionConfig struct {
	ServerURL     string `json:"server_url"`
	CredentialRef string `json:"credential_ref"`
}
type WorkspaceScope struct {
	NamespaceID      domain.ID `json:"namespace_id"`
	DefaultOutcomeID domain.ID `json:"default_outcome_id"`
}
type WorkspaceConfig struct {
	Root   string `json:"root"`
	Format string `json:"format"`
}
type LeaseConfig struct {
	RequestedTTLSeconds int `json:"requested_ttl_seconds"`
}
type OutputConfig struct {
	DefaultFormat string `json:"default_format"`
}

func (c Config) Validate() error {
	if c.SchemaVersion != 1 || c.Kind != "WOSWorkspace" {
		return fmt.Errorf("unsupported workspace schema/kind")
	}
	u, err := url.Parse(c.Connection.ServerURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid server URL")
	}
	if !regexp.MustCompile(`^env:[A-Za-z_][A-Za-z0-9_]*$`).MatchString(c.Connection.CredentialRef) {
		return fmt.Errorf("credential_ref must reference an environment variable")
	}
	if err = c.Scope.NamespaceID.Validate(); err != nil {
		return err
	}
	if err = c.Scope.DefaultOutcomeID.Validate(); err != nil {
		return err
	}
	if c.Workspace.Root != ".wos/work" || c.Workspace.Format != "yaml" {
		return fmt.Errorf("schema 1 workspace root/format must be .wos/work and yaml")
	}
	if c.Lease.RequestedTTLSeconds < 30 || c.Lease.RequestedTTLSeconds > 86400 {
		return fmt.Errorf("requested TTL outside client range")
	}
	if c.Output.DefaultFormat != "text" && c.Output.DefaultFormat != "json" {
		return fmt.Errorf("output format must be text/json")
	}
	return nil
}
func (c Config) DefaultScope() domain.Scope {
	return domain.Scope{NamespaceID: c.Scope.NamespaceID, OutcomeID: c.Scope.DefaultOutcomeID}
}

type ContractDocument struct {
	SchemaVersion int                 `json:"schema_version"`
	Kind          string              `json:"kind"`
	Contract      domain.WorkContract `json:"contract"`
}
type CheckpointDraft struct {
	SchemaVersion int                                 `json:"schema_version"`
	Kind          string                              `json:"kind"`
	ContractID    domain.ID                           `json:"contract_id"`
	SpecDigest    string                              `json:"spec_digest"`
	Checkpoint    application.ContractCheckpointInput `json:"checkpoint"`
}
type ResultDraft struct {
	SchemaVersion         int                                 `json:"schema_version"`
	Kind                  string                              `json:"kind"`
	ContractID            domain.ID                           `json:"contract_id"`
	SpecDigest            string                              `json:"spec_digest"`
	Material              domain.WorkResultMaterial           `json:"material"`
	Artifacts             []application.SyncArtifactInput     `json:"artifacts"`
	Evidence              []application.SyncEvidenceInput     `json:"evidence"`
	EvidenceLinks         []application.SyncEvidenceLinkInput `json:"evidence_links"`
	CriterionEvidenceKeys []CriterionEvidenceKeys             `json:"criterion_evidence_keys"`
}
type CriterionEvidenceKeys struct {
	CriterionID       domain.ID                `json:"criterion_id"`
	CriterionRevision domain.CriterionRevision `json:"criterion_revision"`
	EvidenceKeys      []string                 `json:"evidence_keys"`
}
