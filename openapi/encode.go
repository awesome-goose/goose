package openapi

import (
	"bytes"
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// JSON is the document as indented JSON with sorted keys.
func (d Document) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]any(d)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// YAML is the document as YAML with sorted keys.
func (d Document) YAML() ([]byte, error) {
	// through JSON so numbers and the order of keys come out the same everywhere
	j, err := json.Marshal(map[string]any(d))
	if err != nil {
		return nil, err
	}
	var generic yaml.Node
	var v any
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if err := generic.Encode(v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&generic); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
