// Package woscli implements an API-only client workspace. It never executes
// instructions, shell expressions, artifact contents or agent runtimes.
package woscli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"go.yaml.in/yaml/v3"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxDocumentBytes = 256 << 10

// YAMLJSON validates the bounded WOS YAML 1.2 profile before typed decoding.
func YAMLJSON(raw []byte) ([]byte, error) {
	if len(raw) > MaxDocumentBytes || !utf8.Valid(raw) {
		return nil, fmt.Errorf("document must be UTF-8 and at most 256 KiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("exactly one YAML document is required")
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("document root must be a mapping")
	}
	count := 0
	v, err := yamlValue(document.Content[0], "$", 0, &count)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func yamlValue(n *yaml.Node, path string, depth int, count *int) (any, error) {
	*count++
	if depth > 16 || *count > 10000 {
		return nil, fmt.Errorf("%s:%d:%d: document depth/node limit", path, n.Line, n.Column)
	}
	fail := func(message string) (any, error) {
		return nil, fmt.Errorf("%s:%d:%d: %s", path, n.Line, n.Column, message)
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode || n.Style&yaml.TaggedStyle != 0 {
		return fail("anchors, aliases and explicit tags are forbidden")
	}
	switch n.Kind {
	case yaml.MappingNode:
		out := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
				return fail("mapping keys must be strings; merge keys are forbidden")
			}
			if _, ok := out[key.Value]; ok {
				return fail("duplicate key " + key.Value)
			}
			if key.Anchor != "" || key.Style&yaml.TaggedStyle != 0 {
				return fail("tagged/anchored key forbidden")
			}
			v, err := yamlValue(n.Content[i+1], path+"."+key.Value, depth+1, count)
			if err != nil {
				return nil, err
			}
			out[key.Value] = v
		}
		return out, nil
	case yaml.SequenceNode:
		out := []any{}
		for i, child := range n.Content {
			v, err := yamlValue(child, fmt.Sprintf("%s[%d]", path, i), depth+1, count)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		if len(n.Value) > 65536 {
			return fail("scalar exceeds 64 KiB")
		}
		switch n.Tag {
		case "!!str", "!!timestamp":
			return n.Value, nil
		case "!!null":
			if n.Value != "null" && n.Value != "~" && n.Value != "" {
				return fail("invalid null")
			}
			return nil, nil
		case "!!bool":
			if n.Value != "true" && n.Value != "false" {
				return fail("boolean must be true or false")
			}
			return n.Value == "true", nil
		case "!!int", "!!float":
			// Decimal JSON grammar deliberately excludes YAML octal/hex, infinities and underscores.
			if !json.Valid([]byte(n.Value)) || strings.ContainsAny(n.Value, "_xXoO") {
				return fail("number must use decimal JSON syntax")
			}
			var number json.Number
			d := json.NewDecoder(strings.NewReader(n.Value))
			d.UseNumber()
			if err := d.Decode(&number); err != nil {
				return fail("invalid number")
			}
			if _, err := strconv.ParseFloat(number.String(), 64); err != nil {
				return fail("number outside supported range")
			}
			return number, nil
		default:
			return fail("unsupported YAML tag")
		}
	}
	return fail("unsupported YAML node")
}
func DecodeDocument(raw []byte, target any) error {
	data, err := YAMLJSON(raw)
	if err != nil {
		return err
	}
	data, err = commands.Normalize(data, reflect.TypeOf(target).Elem())
	if err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(target); err != nil {
		return fmt.Errorf("document schema: %w", err)
	}
	return nil
}
func EncodeDocument(value any) ([]byte, error) {
	raw, err := commands.Encode(value)
	if err != nil {
		return nil, err
	}
	var node yaml.Node
	if err = yaml.Unmarshal(raw, &node); err != nil {
		return nil, err
	}
	var clearStyle func(*yaml.Node)
	clearStyle = func(n *yaml.Node) {
		n.Style = 0
		for _, child := range n.Content {
			clearStyle(child)
		}
	}
	clearStyle(&node)
	var out bytes.Buffer
	e := yaml.NewEncoder(&out)
	e.SetIndent(2)
	if err = e.Encode(&node); err != nil {
		return nil, err
	}
	if err = e.Close(); err != nil {
		return nil, err
	}
	if out.Len() > MaxDocumentBytes {
		return nil, fmt.Errorf("document exceeds 256 KiB")
	}
	return out.Bytes(), nil
}
