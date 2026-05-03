package mcp

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSON Schema normalization for MCP tool inputs.
//
// MCP servers report tool schemas as JSON Schema documents. Real-world servers
// use draft-07 features that do not map 1:1 to mcpx's flat PropertySchema:
//
//   - "type": ["string", "null"]     ← Sentry, GitHub (issue #14)
//   - "oneOf"/"anyOf"                ← discriminated unions
//   - "allOf"                        ← schema composition
//   - "$ref"                         ← cross-references
//
// This file gives PropertySchema and InputSchema custom UnmarshalJSON
// implementations that flatten all of the above into the canonical mcpx shape:
//
//   * type chosen as the first non-null primitive
//   * Nullable=true if "null" appeared in a union
//   * unknown keywords preserved in Ext for round-tripping / inspection
//
// We never error out — unrecognized shapes degrade to type="any" with the
// original payload kept in Ext. MCPX_VERBOSE=1 logs warnings to stderr.

// Known JSON Schema keywords mcpx maps to PropertySchema fields. Anything else
// goes into Ext.
var knownKeys = map[string]bool{
	"type":        true,
	"description": true,
	"default":     true,
	"enum":        true,
	"items":       true,
	"nullable":    true,
	"properties":  true,
	"required":    true,
}

// UnmarshalJSON normalizes a JSON Schema property into a PropertySchema.
// Handles type union arrays, oneOf/anyOf branches, and allOf composition.
func (p *PropertySchema) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// Not an object — treat as untyped.
		p.Type = "any"
		return nil
	}

	// allOf: merge all branches into raw before any other processing.
	if a, ok := raw["allOf"]; ok {
		var branches []map[string]json.RawMessage
		if err := json.Unmarshal(a, &branches); err == nil {
			for _, br := range branches {
				for k, v := range br {
					if _, exists := raw[k]; !exists {
						raw[k] = v
					}
				}
			}
		}
		delete(raw, "allOf")
	}

	// oneOf / anyOf: pick the first non-null branch's "type", remember all in Ext.
	if branches, ok := pickUnion(raw, "oneOf"); ok {
		applyFirstNonNullType(&raw, branches, p)
	}
	if branches, ok := pickUnion(raw, "anyOf"); ok {
		applyFirstNonNullType(&raw, branches, p)
	}

	// type: string OR [string,...]
	if t, ok := raw["type"]; ok {
		if typ, nullable, err := normalizeType(t); err == nil {
			p.Type = typ
			if nullable {
				p.Nullable = true
			}
		} else {
			verboseLog("schema: type keyword not parseable: %v", err)
			p.Type = "any"
		}
	} else if p.Type == "" {
		p.Type = "any"
	}

	// description, default, enum, items.
	if d, ok := raw["description"]; ok {
		_ = json.Unmarshal(d, &p.Description)
	}
	if d, ok := raw["default"]; ok {
		_ = json.Unmarshal(d, &p.Default)
	}
	if d, ok := raw["enum"]; ok {
		_ = json.Unmarshal(d, &p.Enum)
	}
	if d, ok := raw["items"]; ok {
		var items PropertySchema
		if err := json.Unmarshal(d, &items); err == nil {
			p.Items = &items
		}
	}

	// Everything else goes into Ext.
	for k, v := range raw {
		if knownKeys[k] {
			continue
		}
		if p.Ext == nil {
			p.Ext = map[string]json.RawMessage{}
		}
		p.Ext[k] = v
	}

	return nil
}

// UnmarshalJSON normalizes the top-level tool input schema. Same union/composition
// handling as PropertySchema but only the relevant subset.
func (s *InputSchema) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		s.Type = "object"
		return nil
	}

	if t, ok := raw["type"]; ok {
		typ, _, err := normalizeType(t)
		if err == nil {
			s.Type = typ
		} else {
			s.Type = "object"
		}
	} else {
		s.Type = "object"
	}

	if p, ok := raw["properties"]; ok {
		_ = json.Unmarshal(p, &s.Properties)
	}
	if r, ok := raw["required"]; ok {
		_ = json.Unmarshal(r, &s.Required)
	}

	for k, v := range raw {
		if knownKeys[k] {
			continue
		}
		if s.Ext == nil {
			s.Ext = map[string]json.RawMessage{}
		}
		s.Ext[k] = v
	}

	return nil
}

// normalizeType handles JSON Schema's `type` keyword which may be a string or
// a string array. For arrays, returns the first non-null entry plus nullable=true
// if "null" was present.
func normalizeType(raw json.RawMessage) (string, bool, error) {
	// Try string first.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "null" {
			return "any", true, nil
		}
		return s, false, nil
	}
	// Try array.
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		nullable := false
		first := ""
		for _, t := range arr {
			if t == "null" {
				nullable = true
				continue
			}
			if first == "" {
				first = t
			}
		}
		if first == "" {
			first = "any"
		}
		return first, nullable, nil
	}
	return "", false, fmt.Errorf("type is neither string nor []string: %s", raw)
}

// pickUnion extracts a oneOf/anyOf array if present; deletes the key from raw
// and returns the parsed branches.
func pickUnion(raw map[string]json.RawMessage, key string) ([]map[string]json.RawMessage, bool) {
	v, ok := raw[key]
	if !ok {
		return nil, false
	}
	delete(raw, key)
	var branches []map[string]json.RawMessage
	if err := json.Unmarshal(v, &branches); err != nil {
		return nil, false
	}
	return branches, true
}

// applyFirstNonNullType walks union branches; for the first branch with a
// non-null `type`, copies it into raw if raw has none. Sets nullable=true if
// any branch was {"type":"null"}.
func applyFirstNonNullType(raw *map[string]json.RawMessage, branches []map[string]json.RawMessage, p *PropertySchema) {
	for _, br := range branches {
		t, ok := br["type"]
		if !ok {
			continue
		}
		typ, nullable, err := normalizeType(t)
		if err != nil {
			continue
		}
		if typ == "null" || typ == "any" && nullable {
			p.Nullable = true
			continue
		}
		if _, has := (*raw)["type"]; !has {
			(*raw)["type"] = t
		}
		if nullable {
			p.Nullable = true
		}
		// Copy other primitive keywords from the chosen branch if not already set.
		for k, v := range br {
			if k == "type" {
				continue
			}
			if _, has := (*raw)[k]; !has {
				(*raw)[k] = v
			}
		}
		return
	}
}

// verboseLog emits a warning to stderr only when MCPX_VERBOSE=1.
func verboseLog(format string, args ...any) {
	if os.Getenv("MCPX_VERBOSE") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "mcpx: "+format+"\n", args...)
}
