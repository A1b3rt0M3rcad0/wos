package wossdk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/url"
)

// CommandReceipt is a durable commit observation. ResultOmitted distinguishes a
// compact commit receipt from a full stored response; neither a state GET nor an
// HTTP status alone substitutes for this observation.
type CommandReceipt struct {
	CommandID             domain.ID              `json:"command_id"`
	OutcomeRevision       domain.OutcomeRevision `json:"outcome_revision"`
	ResponseJSON          json.RawMessage        `json:"response_json,omitempty"`
	ResultOmitted         bool                   `json:"result_omitted,omitempty"`
	DeliveryID            *domain.ID             `json:"delivery_id,omitempty"`
	DeliveryDigest        string                 `json:"delivery_digest,omitempty"`
	DeliveryRound         uint64                 `json:"delivery_round,omitempty,string"`
	WithdrawalID          *domain.ID             `json:"withdrawal_id,omitempty"`
	CompletionAuthorityID *domain.ID             `json:"completion_authority_id,omitempty"`
}

// GetCommandReceiptObservation reads the original Principal-bound result, without retrying
// or changing an intention. If the command ID is unknown after a timeout, replay
// the identical original command with its original idempotency key first.
func (c *Client) GetCommandReceiptObservation(ctx context.Context, namespace, commandID domain.ID) (CommandReceipt, error) {
	var value CommandReceipt
	if err := namespace.Validate(); err != nil {
		return value, err
	}
	if err := commandID.Validate(); err != nil {
		return value, err
	}
	err := c.do(ctx, "GET", "/namespaces/"+namespace.String()+"/commands/"+commandID.String(), "", nil, &value)
	if err != nil {
		return value, err
	}
	if value.CommandID != commandID || value.OutcomeRevision == 0 || (!value.ResultOmitted && !json.Valid(value.ResponseJSON)) {
		return CommandReceipt{}, fmt.Errorf("invalid durable command receipt binding")
	}
	return value, nil
}

func (c *Client) Capabilities(ctx context.Context) (application.ExecutionCapabilities, error) {
	var v application.ExecutionCapabilities
	err := c.do(ctx, "GET", "/capabilities", "", nil, &v)
	return v, err
}
func (c *Client) GetWorkContract(ctx context.Context, scope domain.Scope, id domain.ID) (application.ReadResult[application.WorkContractView], error) {
	var v application.ReadResult[application.WorkContractView]
	err := c.do(ctx, "GET", scopePath(scope)+"/work-contracts/"+id.String(), "", nil, &v)
	return v, err
}
func (c *Client) WorkContractHistory(ctx context.Context, scope domain.Scope, f ports.ContractFilter, cursor string) (application.ContractPage, error) {
	var v application.ContractPage
	q := url.Values{"limit": {fmt.Sprint(f.Limit)}, "cursor": {cursor}, "holder_principal_id": {f.HolderPrincipalID}, "work_item_id": {string(f.WorkItemID)}, "status": {string(f.Status)}}
	err := c.do(ctx, "GET", scopePath(scope)+"/work-contracts?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) ContractRecords(ctx context.Context, scope domain.Scope, id domain.ID, section string, limit int, cursor string) (application.ContractRecordPage, error) {
	var v application.ContractRecordPage
	if section != "checkpoints" && section != "submissions" {
		return v, fmt.Errorf("invalid contract section")
	}
	q := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", scopePath(scope)+"/work-contracts/"+id.String()+"/"+section+"?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) GetWorkSubmission(ctx context.Context, scope domain.Scope, id domain.ID) (application.ReadResult[domain.WorkSubmission], error) {
	var v application.ReadResult[domain.WorkSubmission]
	err := c.do(ctx, "GET", scopePath(scope)+"/work-submissions/"+id.String(), "", nil, &v)
	return v, err
}
func (c *Client) GetCommandReceipt(ctx context.Context, ns, id domain.ID) (domain.StoredCommandResult, error) {
	v, err := c.GetCommandReceiptObservation(ctx, ns, id)
	if err != nil {
		return domain.StoredCommandResult{}, err
	}
	if v.ResultOmitted {
		return domain.StoredCommandResult{}, fmt.Errorf("command %s committed but its full response was omitted; use GetCommandReceiptObservation for the compact receipt", id)
	}
	return domain.StoredCommandResult{CommandID: v.CommandID, OutcomeRevision: v.OutcomeRevision, ResponseJSON: v.ResponseJSON}, nil
}

func (c *Client) ListAvailableWork(ctx context.Context, scope domain.Scope, limit int, cursor string) (application.AvailableWorkPage, error) {
	var v application.AvailableWorkPage
	q := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", scopePath(scope)+"/available-work?"+q.Encode(), "", nil, &v)
	return v, err
}

func (c *Client) GetNamespaceWorkProtocol(ctx context.Context, ns domain.ID) (domain.NamespaceWorkProtocol, error) {
	var v domain.NamespaceWorkProtocol
	err := c.do(ctx, "GET", "/namespaces/"+ns.String()+"/work-protocol", "", nil, &v)
	return v, err
}

func (c *Client) GetWorkCheckpoint(ctx context.Context, scope domain.Scope, id domain.ID) (application.ReadResult[domain.WorkCheckpoint], error) {
	var v application.ReadResult[domain.WorkCheckpoint]
	err := c.do(ctx, "GET", scopePath(scope)+"/work-checkpoints/"+id.String(), "", nil, &v)
	return v, err
}
