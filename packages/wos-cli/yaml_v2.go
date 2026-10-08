package woscli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"go.yaml.in/yaml/v3"
	"io"
	"reflect"
	"regexp"
	"unicode/utf8"
)

const MaxLocalDocumentV2 = 1 << 20

var technicalV2Pattern = regexp.MustCompile(`^\$\._local\.(pending_operations\[[0-9]+\]\.(payload|response)|pending\.(envelope\.payload|response))$`)

func technicalV2Path(path string) bool { return technicalV2Pattern.MatchString(path) }
func validTechnicalV2Blob(value string) bool {
	if len(value) > base64.StdEncoding.EncodedLen(signing.MaxEnvelopeBytes) {
		return false
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(value)
	return e == nil && len(raw) <= signing.MaxEnvelopeBytes && base64.StdEncoding.EncodeToString(raw) == value
}
func YAMLJSONV2(raw []byte) ([]byte, error) {
	if len(raw) > MaxLocalDocumentV2 || !utf8.Valid(raw) {
		return nil, fmt.Errorf("local v2 document must be UTF-8 and at most 1 MiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var doc yaml.Node
	if e := decoder.Decode(&doc); e != nil {
		return nil, e
	}
	if e := decoder.Decode(new(yaml.Node)); e != io.EOF {
		return nil, fmt.Errorf("exactly one YAML document is required")
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("local document root must be a mapping")
	}
	count := 0
	value, e := yamlValueLimited(doc.Content[0], "$", 0, &count, true)
	if e != nil {
		return nil, e
	}
	return json.Marshal(value)
}
func presentationShape(value any, path string) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			next := path + "." + key
			if scalar, ok := child.(string); ok && len(scalar) > signing.MaxScalarBytes && technicalV2Path(next) {
				node[key] = ""
			} else {
				presentationShape(child, next)
			}
		}
	case []any:
		for i, child := range node {
			presentationShape(child, fmt.Sprintf("%s[%d]", path, i))
		}
	}
}
func DecodeV2Document(raw []byte, target any) error {
	data, e := YAMLJSONV2(raw)
	if e != nil {
		return e
	}
	typed := reflect.TypeOf(target)
	if typed == nil || typed.Kind() != reflect.Pointer {
		return fmt.Errorf("document destination must be a pointer")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var shape any
	if e = decoder.Decode(&shape); e != nil {
		return e
	}
	// Only already-bounded technical blobs are abbreviated for the strict field
	// validator. The original bytes are then decoded; no proof bytes are modified.
	presentationShape(shape, "$")
	bounded, e := json.Marshal(shape)
	if e != nil {
		return e
	}
	if e = signing.DecodeStrict(bounded, reflect.New(typed.Elem()).Interface(), MaxLocalDocumentV2); e != nil {
		return e
	}
	return commands.Decode(data, target)
}
func EncodeV2Document(value any) ([]byte, error) {
	raw, e := json.Marshal(value)
	if e != nil {
		return nil, e
	}
	if len(raw) > MaxLocalDocumentV2 {
		return nil, fmt.Errorf("local document exceeds 1 MiB")
	}
	var node yaml.Node
	if e = yaml.Unmarshal(raw, &node); e != nil {
		return nil, e
	}
	var styles func(*yaml.Node)
	styles = func(n *yaml.Node) {
		n.Style = 0
		for _, c := range n.Content {
			styles(c)
		}
	}
	styles(&node)
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if e = encoder.Encode(&node); e != nil {
		return nil, e
	}
	if e = encoder.Close(); e != nil {
		return nil, e
	}
	if _, e = YAMLJSONV2(out.Bytes()); e != nil {
		return nil, e
	}
	return out.Bytes(), nil
}
