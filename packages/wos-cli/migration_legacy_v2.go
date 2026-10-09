package woscli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (file LegacyUnsignedFileV2) MarshalJSON() ([]byte, error) {
	type plain LegacyUnsignedFileV2
	raw, e := json.Marshal(plain(file))
	if e != nil {
		return nil, e
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var tree any
	if e = decoder.Decode(&tree); e != nil {
		return nil, e
	}
	quoteLegacyLocalCountersV2(tree)
	return json.Marshal(tree)
}
func quoteLegacyLocalCountersV2(tree any) {
	switch node := tree.(type) {
	case map[string]any:
		for key, value := range node {
			switch key {
			case "version", "lease_version", "fencing_token", "revision", "criterion_revision", "work_item_version_at_acquire", "outcome_revision_at_acquire", "work_item_version", "outcome_revision":
				if number, ok := value.(json.Number); ok {
					node[key] = number.String()
					continue
				}
			}
			if key != "external_context" && key != "payload" {
				quoteLegacyLocalCountersV2(value)
			}
		}
	case []any:
		for _, value := range node {
			quoteLegacyLocalCountersV2(value)
		}
	}
}
func readLegacyUnsignedV2(w *Workspace, profile ProfileV2, id d.ID) (LegacyUnsignedFileV2, bool, error) {
	var file LegacyUnsignedFileV2
	if id.Validate() != nil {
		return file, false, fmt.Errorf("valid legacy contract ID required")
	}
	path, e := w.DocumentPath(filepath.Join(".wos/profiles", profile.Name, "contract", id.String()))
	if e != nil {
		return file, false, e
	}
	raw, e := w.ReadV2(path)
	if e != nil {
		return file, false, e
	}
	return decodeLegacyUnsignedV2(raw, profile, id)
}
func decodeLegacyUnsignedV2(raw []byte, profile ProfileV2, id d.ID) (LegacyUnsignedFileV2, bool, error) {
	var file LegacyUnsignedFileV2
	js, e := YAMLJSONV2(raw)
	if e != nil {
		return file, false, e
	}
	var header struct {
		Kind string `json:"kind"`
	}
	if e = json.Unmarshal(js, &header); e != nil {
		return file, false, e
	}
	if header.Kind != "WOSLegacyUnsignedContract" {
		return file, false, nil
	}
	if e = DecodeV2Document(raw, &file); e != nil {
		return file, true, e
	}
	c := file.Contract.Contract
	if file.SchemaVersion != 2 || file.Protocol != "legacy_unsigned" || file.SourceOrigin != profile.Binding.ServerOrigin || file.SourceNamespaceID != profile.Binding.NamespaceID || c.ID != id || c.Scope.NamespaceID != profile.Binding.NamespaceID || c.SignedBinding != nil || c.Validate() != nil {
		return file, true, fmt.Errorf("legacy unsigned record binding differs")
	}
	return file, true, nil
}
func legacyUnsignedSummaryV2(file LegacyUnsignedFileV2) any {
	c := file.Contract.Contract
	return map[string]any{"contract_id": c.ID, "work_item_id": c.WorkItemID, "outcome_id": c.Scope.OutcomeID, "title": c.Spec.Title, "protocol": "legacy_unsigned", "snapshot_status": c.Status, "signed_authority": false}
}
