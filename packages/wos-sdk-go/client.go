// Package wossdk is an HTTP client for WOS. Mutations never retry automatically.
package wossdk

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL string
	Prefix  string
	Token   string
	HTTP    *http.Client
}

func New(base, token string, client *http.Client) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid WOS URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	safeClient := *client
	safeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{BaseURL: strings.TrimRight(base, "/"), Prefix: "/api/v1", Token: token, HTTP: &safeClient}, nil
}

type Error struct {
	Status        int
	Code          string
	Message       string
	Retryable     bool
	CorrelationID string
	Details       *domain.ErrorDetails
}

func (e *Error) Error() string { return fmt.Sprintf("WOS %s (%d): %s", e.Code, e.Status, e.Message) }

type CommandResult[T any] struct {
	Value                 T                             `json:"value"`
	OutcomeRevision       domain.OutcomeRevision        `json:"outcome_revision"`
	CommandID             domain.ID                     `json:"command_id"`
	IdempotentReplay      bool                          `json:"idempotent_replay"`
	ResultOmitted         bool                          `json:"result_omitted"`
	CreatedRefs           map[string]domain.EntityRef   `json:"created_refs,omitempty"`
	CreatedItems          []WorkspaceCreatedItemReceipt `json:"created_items,omitempty"`
	EntityID              *domain.ID                    `json:"entity_id,omitempty"`
	EvidenceID            *domain.ID                    `json:"evidence_id,omitempty"`
	ArtifactID            *domain.ID                    `json:"artifact_id,omitempty"`
	ObservationID         *domain.ID                    `json:"observation_id,omitempty"`
	DecisionID            *domain.ID                    `json:"decision_id,omitempty"`
	RelationID            *domain.ID                    `json:"relation_id,omitempty"`
	EvidenceLinkIDs       []domain.ID                   `json:"evidence_link_ids,omitempty"`
	DeliveryID            *domain.ID                    `json:"delivery_id,omitempty"`
	DeliveryDigest        string                        `json:"delivery_digest,omitempty"`
	DeliveryRound         uint64                        `json:"delivery_round,omitempty,string"`
	WithdrawalID          *domain.ID                    `json:"withdrawal_id,omitempty"`
	CompletionAuthorityID *domain.ID                    `json:"completion_authority_id,omitempty"`
}

type WorkspaceCreatedItemReceipt struct {
	LocalKey     string           `json:"local_key"`
	Ref          domain.EntityRef `json:"ref"`
	Version      domain.Version   `json:"version,string"`
	CriterionIDs []domain.ID      `json:"criterion_ids"`
}

func NewIdempotencyKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func (c *Client) do(ctx context.Context, method, path, key string, input, out any) error {
	return c.doBounded(ctx, method, path, key, input, out, 256<<10)
}

// Technical expansions have explicit bounds distinct from ordinary responses.
func (c *Client) doBounded(ctx context.Context, method, path, key string, input, out any, maximum int64) error {
	var body io.Reader
	if input != nil {
		b, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+c.Prefix+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// net/http treats Idempotency-Key as permission to retry a request after
	// a reused connection fails. Mutations remain explicitly reconciled by
	// callers; remove the replay body so the standard transport cannot resend.
	if method != http.MethodGet && method != http.MethodHead {
		req.GetBody = nil
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maximum+1))
	if err != nil {
		return err
	}
	if int64(len(raw)) > maximum {
		return fmt.Errorf("WOS response exceeds %d bytes", maximum)
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return &Error{Status: resp.StatusCode, Code: "transport_redirect", Message: "redirect refused; restore the exact trusted destination"}
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error struct {
				Code      string               `json:"code"`
				Message   string               `json:"message"`
				Retryable bool                 `json:"retryable"`
				Details   *domain.ErrorDetails `json:"details,omitempty"`
			}
		}
		json.Unmarshal(raw, &e)
		return &Error{Status: resp.StatusCode, Code: e.Error.Code, Message: e.Error.Message, Retryable: e.Error.Retryable, CorrelationID: resp.Header.Get("X-Correlation-ID"), Details: e.Error.Details}
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}
func command[T any](ctx context.Context, c *Client, name, key string, cmd any) (CommandResult[T], error) {
	var result CommandResult[T]
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return result, err
	}
	raw, err := commands.Encode(cmd)
	if err != nil {
		return result, err
	}
	maximum := int64(256 << 10)
	switch cmd.(type) {
	case application.AppendWorkActivityCommand, application.AcquireSignedWorkContractCommand, application.AcquireNextSignedWorkContractCommand, application.RenewSignedWorkContractCommand, application.ResumeSignedWorkContractCommand, application.ReturnSignedWorkCommand, application.AcquireSignedReviewContractCommand, application.AcquireNextSignedReviewContractCommand, application.RenewSignedReviewContractCommand, application.ResumeSignedReviewContractCommand, application.ReturnSignedReviewCommand, application.InterveneSignedReviewCaseCommand, application.RevokeSignedContractCommand:
		maximum = domain.MaxSignedOperationResultBytes + (64 << 10)
	}
	err = c.doBounded(ctx, "POST", "/commands/"+name, key, map[string]any{"command": json.RawMessage(raw)}, &result, maximum)
	return result, err
}
func scopePath(scope domain.Scope) string {
	return "/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String()
}
func (c *Client) GetContinuity(ctx context.Context, scope domain.Scope, limit int) (application.ContinuitySnapshot, error) {
	var v application.ContinuitySnapshot
	err := c.do(ctx, "GET", scopePath(scope)+fmt.Sprintf("/continuity?limit=%d", limit), "", nil, &v)
	return v, err
}
func (c *Client) GetContinuitySection(ctx context.Context, scope domain.Scope, section string, limit int, cursor string) (application.SectionPage, error) {
	var v application.SectionPage
	q := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", scopePath(scope)+"/continuity/"+url.PathEscape(section)+"?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) SearchOutcomes(ctx context.Context, namespace domain.ID, text string, limit int, cursor string) (application.OutcomePage, error) {
	var v application.OutcomePage
	q := url.Values{"text": {text}, "limit": {fmt.Sprint(limit)}, "cursor": {cursor}}
	err := c.do(ctx, "GET", "/namespaces/"+namespace.String()+"/outcomes?"+q.Encode(), "", nil, &v)
	return v, err
}
func (c *Client) GetWorkContext(ctx context.Context, scope domain.Scope, id domain.ID) (application.WorkContext, error) {
	var v application.WorkContext
	err := c.do(ctx, "GET", scopePath(scope)+"/work-items/"+id.String()+"/context", "", nil, &v)
	return v, err
}
