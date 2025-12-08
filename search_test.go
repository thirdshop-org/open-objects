package main

import (
	"testing"
)

func TestParseSearchPropExact(t *testing.T) {
	c, err := ParseSearchProp("name:SKF")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c.PropName != "name" {
		t.Errorf("PropName expected 'name', got '%s'", c.PropName)
	}
	if c.IsRange {
		t.Errorf("expected IsRange=false")
	}
	if c.ExactVal != "SKF" {
		t.Errorf("ExactVal expected 'SKF', got '%s'", c.ExactVal)
	}
}

func TestParseSearchPropRange(t *testing.T) {
	c, err := ParseSearchProp("d_int:10.5..20")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !c.IsRange {
		t.Fatalf("expected IsRange=true")
	}
	if c.MinVal != 10.5 || c.MaxVal != 20 {
		t.Errorf("expected min=10.5 max=20, got %f %f", c.MinVal, c.MaxVal)
	}
}

func TestParseSearchPropInvalid(t *testing.T) {
	tests := []string{
		"wrongformat",
		"name:",
		"d_int:10..abc",
		"d_int:abc..10",
	}
	for _, input := range tests {
		if _, err := ParseSearchProp(input); err == nil {
			t.Errorf("expected error for '%s'", input)
		}
	}
}

func TestMatchesCriteriaExact(t *testing.T) {
	c, _ := ParseSearchProp("name:SKF")
	if !c.MatchesCriteria("SKF") {
		t.Errorf("expected match for same string")
	}
	if c.MatchesCriteria("SKG") {
		t.Errorf("did not expect match for different string")
	}
	if c.MatchesCriteria(12) {
		t.Errorf("did not expect match for different type/value")
	}
}

func TestMatchesCriteriaRange(t *testing.T) {
	c, _ := ParseSearchProp("d_int:10..20")

	ok := c.MatchesCriteria(10)
	if !ok {
		t.Errorf("expected 10 to match range")
	}
	ok = c.MatchesCriteria(15.5)
	if !ok {
		t.Errorf("expected 15.5 to match range")
	}
	ok = c.MatchesCriteria(20)
	if !ok {
		t.Errorf("expected 20 to match range")
	}
	if c.MatchesCriteria(9.9) {
		t.Errorf("did not expect 9.9 to match range")
	}
	if c.MatchesCriteria("abc") {
		t.Errorf("did not expect non numeric to match range")
	}
}
