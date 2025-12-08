package main

import (
	"database/sql"
	"encoding/json"
	"testing"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		t.Fatalf("migrations: %v", err)
	}

	return db
}

func TestCmdAddCreatesPartWithLocationAndNormalization(t *testing.T) {
	seedTemplates()
	db := newTestDB(t)
	defer db.Close()

	loc, err := CreateLocation(db, "Boite Roulements", nil, "BOX", "")
	if err != nil {
		t.Fatalf("create location: %v", err)
	}

	args := []string{
		"--type=bearing",
		"--name=Roulement 6001",
		`--props={"d_int":"1cm","d_ext":32,"width":10}`,
		"--loc=Boite Roulements",
	}

	if err := cmdAdd(db, args); err != nil {
		t.Fatalf("cmdAdd returned error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count); err != nil {
		t.Fatalf("count parts: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 part inserted, got %d", count)
	}

	var typeName, name, propsJSON string
	var locID sql.NullInt64
	if err := db.QueryRow("SELECT type, name, props, location_id FROM parts LIMIT 1").
		Scan(&typeName, &name, &propsJSON, &locID); err != nil {
		t.Fatalf("fetch part: %v", err)
	}

	if typeName != "bearing" {
		t.Fatalf("expected type 'bearing', got '%s'", typeName)
	}
	if name != "Roulement 6001" {
		t.Fatalf("expected name 'Roulement 6001', got '%s'", name)
	}
	if !locID.Valid || int(locID.Int64) != loc.ID {
		t.Fatalf("expected location_id %d, got %+v", loc.ID, locID)
	}

	var props map[string]interface{}
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		t.Fatalf("parse props: %v", err)
	}

	if v, ok := props["d_int"].(float64); !ok || v != 10.0 {
		t.Fatalf("expected d_int=10.0 after normalization, got %v", props["d_int"])
	}
	if v, ok := props["d_ext"].(float64); !ok || v != 32.0 {
		t.Fatalf("expected d_ext=32, got %v", props["d_ext"])
	}
	if v, ok := props["width"].(float64); !ok || v != 10.0 {
		t.Fatalf("expected width=10, got %v", props["width"])
	}
}

func TestCmdAddRejectsUnknownType(t *testing.T) {
	seedTemplates()
	db := newTestDB(t)
	defer db.Close()

	err := cmdAdd(db, []string{"--type=unknown", "--name=Invalid"})
	if err == nil {
		t.Fatalf("expected error for unknown type")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM parts").Scan(&count); err != nil {
		t.Fatalf("count parts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no parts inserted on error, got %d", count)
	}
}
