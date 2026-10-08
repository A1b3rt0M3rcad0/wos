package wossdk

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
)

func (c *Client) SigningMutation(ctx context.Context, key string, intent application.SigningSecurityIntent) (application.SigningSecurityResult, error) {
	var result application.SigningSecurityResult
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return result, err
	}
	raw, err := commands.Encode(intent)
	if err != nil {
		return result, err
	}
	err = c.do(ctx, http.MethodPost, "/security/signing-commands", key, json.RawMessage(raw), &result)
	return result, err
}

func (c *Client) SigningIdentity(ctx context.Context, enrollment *domain.ID) (application.SigningIdentityView, error) {
	var result application.SigningIdentityView
	path := "/security/signing-identity"
	if enrollment != nil {
		if err := enrollment.Validate(); err != nil {
			return result, err
		}
		path += "?enrollment_id=" + enrollment.String()
	}
	err := c.do(ctx, http.MethodGet, path, "", nil, &result)
	return result, err
}
