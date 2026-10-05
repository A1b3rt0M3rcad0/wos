package domain

import (
	"net/url"
	"strings"
	"time"
)

// ExternalReference locates shared work; it never grants access or owns the Outcome.
type ExternalReference struct {
	Provider   string `json:"provider"`
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id"`
	URL        string `json:"url,omitempty"`
}

func (r ExternalReference) Validate() error {
	for _, value := range []string{r.Provider, r.Kind, r.ExternalID} {
		if strings.TrimSpace(value) == "" || len(value) > 256 || strings.ContainsRune(value, '\x00') {
			return NewError(ErrorCodeInvalidArgument, "external reference requires provider, kind and identifier within 256 bytes")
		}
	}
	if r.URL != "" {
		u, err := url.Parse(r.URL)
		if err != nil || len(r.URL) > 2048 || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return NewError(ErrorCodeInvalidArgument, "external reference URL must use HTTP(S) without credentials")
		}
	}
	return nil
}
func (o *Outcome) RecordExternalReferenceChange(now time.Time) error {
	if o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "archived outcome context is immutable")
	}
	return o.touch(now)
}
