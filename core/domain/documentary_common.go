package domain

import "strings"

// ExternalReference identifies an external source without implying that WOS
// fetched, authenticated or verified it.
type ExternalReference struct {
	Provider    string `json:"provider"`
	ID          string `json:"id,omitempty"`
	URI         string `json:"uri,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

func (r ExternalReference) Validate() error {
	if strings.TrimSpace(r.Provider) == "" {
		return NewError(ErrorCodeInvalidArgument, "external reference provider is required")
	}
	if strings.TrimSpace(r.ID) == "" && strings.TrimSpace(r.URI) == "" {
		return NewError(ErrorCodeInvalidArgument, "external reference requires id or uri")
	}
	return nil
}

func normalizedUniqueStrings(values []string, field string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return nil, NewError(ErrorCodeInvalidArgument, field+" cannot contain empty values")
		}
		if _, ok := seen[value]; ok {
			return nil, NewError(ErrorCodeInvalidArgument, field+" cannot contain duplicates")
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}
