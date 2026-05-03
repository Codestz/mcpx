package mcp

import (
	"encoding/json"
	"testing"
)

// TestSentryUnionType reproduces issue #14: type as ["string", "null"].
func TestSentryUnionType(t *testing.T) {
	raw := []byte(`{"type": ["string", "null"], "description": "An optional id"}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if p.Type != "string" {
		t.Errorf("type: got %q want %q", p.Type, "string")
	}
	if !p.Nullable {
		t.Error("expected Nullable=true")
	}
	if p.Description != "An optional id" {
		t.Errorf("description: got %q", p.Description)
	}
}

func TestPlainStringType(t *testing.T) {
	raw := []byte(`{"type": "string", "description": "x"}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "string" || p.Nullable {
		t.Errorf("got type=%q nullable=%v", p.Type, p.Nullable)
	}
}

func TestArrayItems(t *testing.T) {
	raw := []byte(`{"type": "array", "items": {"type": "string"}}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "array" {
		t.Fatalf("type: got %q", p.Type)
	}
	if p.Items == nil || p.Items.Type != "string" {
		t.Errorf("items: %+v", p.Items)
	}
}

func TestOneOfFirstNonNull(t *testing.T) {
	raw := []byte(`{"oneOf": [{"type": "null"}, {"type": "string"}]}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "string" {
		t.Errorf("type: got %q want string", p.Type)
	}
	if !p.Nullable {
		t.Error("expected Nullable=true")
	}
}

func TestAllOfMerges(t *testing.T) {
	raw := []byte(`{"allOf": [{"type": "string"}, {"description": "merged"}]}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "string" {
		t.Errorf("type: got %q want string", p.Type)
	}
	if p.Description != "merged" {
		t.Errorf("description: got %q want merged", p.Description)
	}
}

func TestUnknownKeywordsPreservedInExt(t *testing.T) {
	raw := []byte(`{"type": "string", "format": "uuid", "pattern": "^[0-9a-f-]+$"}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Ext["format"] == nil {
		t.Error("format not in Ext")
	}
	if p.Ext["pattern"] == nil {
		t.Error("pattern not in Ext")
	}
}

func TestInputSchemaUnknownKeywords(t *testing.T) {
	raw := []byte(`{
		"type": "object",
		"properties": {"id": {"type": ["string","null"]}},
		"required": ["id"],
		"additionalProperties": false
	}`)
	var s InputSchema
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s.Type != "object" {
		t.Errorf("type: got %q", s.Type)
	}
	if s.Ext["additionalProperties"] == nil {
		t.Error("additionalProperties not in Ext")
	}
	prop, ok := s.Properties["id"]
	if !ok {
		t.Fatal("id property missing")
	}
	if prop.Type != "string" || !prop.Nullable {
		t.Errorf("id: type=%q nullable=%v", prop.Type, prop.Nullable)
	}
}

// Real-world Sentry-style schema fragment, reproducing #14 in full context.
func TestSentryFullToolSchema(t *testing.T) {
	raw := []byte(`{
		"type": "object",
		"properties": {
			"organization_slug": {"type": "string"},
			"project_slug": {"type": ["string", "null"], "description": "Optional project filter"},
			"environment": {"type": ["string", "null"]},
			"limit": {"type": ["integer", "null"], "default": 10}
		},
		"required": ["organization_slug"]
	}`)
	var s InputSchema
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("Sentry schema failed: %v", err)
	}
	if len(s.Properties) != 4 {
		t.Fatalf("properties: got %d want 4", len(s.Properties))
	}
	for name, want := range map[string]struct {
		typ      string
		nullable bool
	}{
		"organization_slug": {"string", false},
		"project_slug":      {"string", true},
		"environment":       {"string", true},
		"limit":             {"integer", true},
	} {
		got := s.Properties[name]
		if got.Type != want.typ || got.Nullable != want.nullable {
			t.Errorf("%s: got type=%q nullable=%v want %q nullable=%v",
				name, got.Type, got.Nullable, want.typ, want.nullable)
		}
	}
}

// Nested oneOf inside properties (GitHub-style discriminated union).
func TestNestedOneOf(t *testing.T) {
	raw := []byte(`{
		"type": "object",
		"properties": {
			"action": {"oneOf": [{"type": "string", "enum": ["create"]}, {"type": "string", "enum": ["delete"]}]}
		}
	}`)
	var s InputSchema
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	got := s.Properties["action"]
	if got.Type != "string" {
		t.Errorf("type: got %q", got.Type)
	}
}

// Degenerate / unknown shape — must not panic, must not error, falls back to "any".
func TestDegenerateSchema(t *testing.T) {
	raw := []byte(`{"$ref": "#/definitions/Whatever"}`)
	var p PropertySchema
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if p.Type != "any" {
		t.Errorf("type: got %q want any", p.Type)
	}
	if p.Ext["$ref"] == nil {
		t.Error("$ref should be preserved in Ext")
	}
}
