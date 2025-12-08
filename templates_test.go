package main

import "testing"

// helper to seed the in-memory templates map for tests
func seedTemplates() {
	Templates = make(map[string]*Template)
	Templates["bearing"] = &Template{
		Name:        "bearing",
		Description: "Roulement",
		Fields: map[string]FieldDef{
			"d_int": {Required: true, Domain: "dimension", DefaultUnit: "mm"},
			"d_ext": {Required: true, Domain: "dimension", DefaultUnit: "mm"},
			"width": {Required: true, Domain: "dimension", DefaultUnit: "mm"},
			"brand": {Required: false},
			"type":  {Required: false},
		},
		Required: []string{"d_int", "d_ext", "width"},
		Optional: []string{"brand", "type"},
	}
}

func TestTypeExists(t *testing.T) {
	seedTemplates()
	if !TypeExists("bearing") {
		t.Fatalf("expected bearing to exist")
	}
	if TypeExists("unknown") {
		t.Fatalf("expected unknown to not exist")
	}
}

func TestValidatePropsUnknownTypeStrict(t *testing.T) {
	seedTemplates()
	err := ValidateProps("unknown_type", map[string]interface{}{})
	if err == nil {
		t.Fatalf("expected error for unknown type in strict mode")
	}
}

func TestValidatePropsMissingRequired(t *testing.T) {
	seedTemplates()
	props := map[string]interface{}{
		"d_int": 10.0,
		"d_ext": 47.0,
		// width missing
	}
	err := ValidateProps("bearing", props)
	if err == nil {
		t.Fatalf("expected error for missing required field")
	}
}

func TestValidatePropsOK(t *testing.T) {
	seedTemplates()
	props := map[string]interface{}{
		"d_int": 10.0,
		"d_ext": 47.0,
		"width": 14.0,
	}
	if err := ValidateProps("bearing", props); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFieldUnits(t *testing.T) {
	seedTemplates()
	units := GetFieldUnits("bearing")
	if units["d_int"] != "mm" || units["width"] != "mm" {
		t.Fatalf("expected mm for dimension fields, got %+v", units)
	}
	if _, ok := units["brand"]; ok {
		t.Fatalf("expected no unit for non dimensional field")
	}
}
