package main

import (
	"testing"
)

func TestParseValueWithUnitOK(t *testing.T) {
	v, err := ParseValueWithUnit("12.5cm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Value != 12.5 || v.Unit != "cm" || !v.HasUnit {
		t.Fatalf("unexpected parsed value: %+v", v)
	}
}

func TestParseValueWithUnitInvalid(t *testing.T) {
	_, err := ParseValueWithUnit("abc")
	if err == nil {
		t.Fatalf("expected error for invalid number")
	}
}

func TestNormalizeValueWithUnit(t *testing.T) {
	res, err := NormalizeValue("1cm", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != 10 || res.Domain != DomainDimension || res.BaseUnit != "mm" {
		t.Fatalf("unexpected normalize result: %+v", res)
	}
}

func TestNormalizeValueDefaultUnit(t *testing.T) {
	res, err := NormalizeValue("12", "V") // no unit provided, default to volts
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Value != 12 || res.Domain != DomainTension || res.BaseUnit != "V" {
		t.Fatalf("unexpected normalize result: %+v", res)
	}
}

func TestNormalizeValueUnknownUnit(t *testing.T) {
	_, err := NormalizeValue("10 zork", "mm")
	if err == nil {
		t.Fatalf("expected error for unknown unit")
	}
}

func TestGetDefaultUnitForField(t *testing.T) {
	if got := GetDefaultUnitForField("d_int"); got != "mm" {
		t.Fatalf("expected mm for d_int, got %s", got)
	}
	if got := GetDefaultUnitForField("volts"); got != "V" {
		t.Fatalf("expected V for volts, got %s", got)
	}
	if got := GetDefaultUnitForField("rpm"); got != "rpm" {
		t.Fatalf("expected rpm for rpm, got %s", got)
	}
}

func TestNormalizePropsWithDefaults(t *testing.T) {
	props := map[string]interface{}{
		"d_int": "1cm",  // should become 10 (mm)
		"volts": 12,     // default V
		"note":  "text", // unchanged text
		"count": 5,      // no default unit; keep numeric
		"width": "14mm", // already mm
	}
	fieldUnits := map[string]string{
		"d_int": "mm",
		"volts": "V",
		"width": "mm",
	}

	norm, err := NormalizeProps(props, fieldUnits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if norm["d_int"] != 10.0 {
		t.Fatalf("expected d_int=10mm, got %v", norm["d_int"])
	}
	if norm["volts"] != 12.0 {
		t.Fatalf("expected volts=12V, got %v", norm["volts"])
	}
	if norm["width"] != 14.0 {
		t.Fatalf("expected width=14mm, got %v", norm["width"])
	}
	if norm["note"] != "text" {
		t.Fatalf("expected note unchanged, got %v", norm["note"])
	}
	if norm["count"] != 5.0 {
		t.Fatalf("expected count=5, got %v", norm["count"])
	}
}

func TestNormalizePropsTextOnly(t *testing.T) {
	props := map[string]interface{}{
		"reference": "6001ZZ",
		"d_int":     "10mm",
	}
	norm, err := NormalizeProps(props, map[string]string{"d_int": "mm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm["reference"] != "6001ZZ" {
		t.Fatalf("reference should stay unchanged, got %v", norm["reference"])
	}
	if norm["d_int"] != 10.0 {
		t.Fatalf("expected d_int=10, got %v", norm["d_int"])
	}
}
