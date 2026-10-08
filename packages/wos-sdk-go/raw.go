package wossdk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/url"
	"strings"
)

// ExecuteCommand sends an already validated public command without retries.
// It supports external clients adapting the generated command catalog.
func (c *Client) ExecuteCommand(ctx context.Context, name, key string, payload json.RawMessage) (json.RawMessage, error) {
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return nil, err
	}
	if !json.Valid(payload) || len(payload) > 256<<10 {
		return nil, fmt.Errorf("invalid command payload")
	}
	var out json.RawMessage
	err := c.do(ctx, "POST", "/commands/"+url.PathEscape(name), key, map[string]any{"command": payload}, &out)
	return out, err
}
func (c *Client) QueryJSON(ctx context.Context, path string) (json.RawMessage, error) {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "..") || strings.ContainsAny(path, "\\#") {
		return nil, fmt.Errorf("invalid API query path")
	}
	var out json.RawMessage
	err := c.do(ctx, "GET", path, "", nil, &out)
	return out, err
}
