package domain

import (
	"encoding/json"
	"strconv"
)

// Read both historical numeric counters and exact v2 decimal strings. Marshal
// remains unchanged so v1 stored fingerprints and public responses are stable.
func decodeCounter(raw []byte) (uint64, error) {
	text := string(raw)
	if len(text) > 0 && text[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return 0, NewError(ErrorCodeInvalidVersion, "invalid decimal counter")
		}
	}
	if text == "" || len(text) > 20 || len(text) > 1 && text[0] == '0' {
		return 0, NewError(ErrorCodeInvalidVersion, "counter must use canonical decimal notation")
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return 0, NewError(ErrorCodeInvalidVersion, "counter must be unsigned decimal")
		}
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, NewError(ErrorCodeInvalidVersion, "counter exceeds uint64")
	}
	return value, nil
}
func (v *Version) UnmarshalJSON(raw []byte) error {
	value, err := decodeCounter(raw)
	if err == nil {
		*v = Version(value)
	}
	return err
}
func (v *OutcomeRevision) UnmarshalJSON(raw []byte) error {
	value, err := decodeCounter(raw)
	if err == nil {
		*v = OutcomeRevision(value)
	}
	return err
}
func (v *CriterionRevision) UnmarshalJSON(raw []byte) error {
	value, err := decodeCounter(raw)
	if err == nil {
		*v = CriterionRevision(value)
	}
	return err
}

func (w *WorkItem) UnmarshalJSON(raw []byte) error {
	type historical WorkItem
	decoded := historical(*w)
	wire := struct {
		*historical
		Fence json.RawMessage `json:"last_fencing_token"`
	}{historical: &decoded}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	if len(wire.Fence) != 0 {
		value, err := decodeCounter(wire.Fence)
		if err != nil {
			return err
		}
		decoded.LastFencingToken = value
	}
	*w = WorkItem(decoded)
	return nil
}
func (w *WorkLease) UnmarshalJSON(raw []byte) error {
	type historical WorkLease
	decoded := historical(*w)
	wire := struct {
		*historical
		Fence json.RawMessage `json:"fencing_token"`
	}{historical: &decoded}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	if len(wire.Fence) != 0 {
		value, err := decodeCounter(wire.Fence)
		if err != nil {
			return err
		}
		decoded.FencingToken = value
	}
	*w = WorkLease(decoded)
	return nil
}
