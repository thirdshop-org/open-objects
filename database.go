package main

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const dbPath = "recycle.db"

// InitDB ouvre la connexion et exécute les migrations
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	// Migration v1: Table de base
	if err := migrateV1(db); err != nil {
		return err
	}

	// Migration v2: Ajout colonne type
	if err := migrateV2(db); err != nil {
		return err
	}

	// Migration v3: Table des fichiers attachés
	if err := migrateV3(db); err != nil {
		return err
	}

	// Index
	if err := createIndexes(db); err != nil {
		return err
	}

	return nil
}

// migrateV1 crée la table parts si elle n'existe pas
func migrateV1(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			props JSON
		)
	`)
	return err
}

// migrateV2 ajoute la colonne type si elle n'existe pas
func migrateV2(db *sql.DB) error {
	if hasColumn(db, "parts", "type") {
		return nil
	}

	_, err := db.Exec("ALTER TABLE parts ADD COLUMN type TEXT DEFAULT ''")
	return err
}

// migrateV3 crée la table des fichiers attachés
func migrateV3(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			part_id INTEGER NOT NULL,
			filename TEXT NOT NULL,
			filepath TEXT NOT NULL,
			filetype TEXT DEFAULT '',
			filesize INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (part_id) REFERENCES parts(id) ON DELETE CASCADE
		)
	`)
	return err
}

func createIndexes(db *sql.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_parts_name ON parts (name)",
		"CREATE INDEX IF NOT EXISTS idx_parts_type ON parts (type)",
		"CREATE INDEX IF NOT EXISTS idx_attachments_part_id ON attachments (part_id)",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// hasColumn vérifie si une colonne existe dans une table
func hasColumn(db *sql.DB, table, column string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		if name == column {
			return true
		}
	}

	return false
}
