package wossdk

import (
	"context"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/url"
)

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
	var v domain.StoredCommandResult
	err := c.do(ctx, "GET", "/namespaces/"+ns.String()+"/commands/"+id.String(), "", nil, &v)
	return v, err
}

func (c *Client) ListAvailableWork(ctx context.Context, scope domain.Scope, limit int, cursor string) (application.AvailableWorkPage, error) {
	var v application.AvailableWorkPage
	q := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", scopePath(scope)+"/available-work?"+q.Encode(), "", nil, &v)
	return v, err
}
