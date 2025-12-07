package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

const dbPath = "recycle.db"

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			props JSON
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_parts_name ON parts (name)")
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func cmdAdd(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	name := fs.String("name", "", "Nom de la pièce")
	props := fs.String("props", "{}", "Propriétés JSON de la pièce")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("le nom est requis (--name)")
	}

	// Valider que props est du JSON valide
	var jsonCheck map[string]interface{}
	if err := json.Unmarshal([]byte(*props), &jsonCheck); err != nil {
		return fmt.Errorf("props invalide: %v", err)
	}

	result, err := db.Exec("INSERT INTO parts (name, props) VALUES (?, ?)", *name, *props)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	fmt.Printf("✓ Pièce ajoutée [ID: %d]\n", id)
	fmt.Printf("  Nom: %s\n", *name)
	fmt.Printf("  Props: %s\n", *props)

	return nil
}

func cmdList(db *sql.DB) error {
	rows, err := db.Query("SELECT id, name, props FROM parts ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	fmt.Println("┌─────┬────────────────────────────────┬────────────────────────────────────────┐")
	fmt.Println("│ ID  │ Nom                            │ Propriétés                             │")
	fmt.Println("├─────┼────────────────────────────────┼────────────────────────────────────────┤")

	for rows.Next() {
		var id int
		var name string
		var props sql.NullString

		if err := rows.Scan(&id, &name, &props); err != nil {
			return err
		}

		propsStr := "{}"
		if props.Valid {
			propsStr = props.String
		}

		// Tronquer si trop long
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		if len(propsStr) > 38 {
			propsStr = propsStr[:35] + "..."
		}

		fmt.Printf("│ %-3d │ %-30s │ %-38s │\n", id, name, propsStr)
		count++
	}

	fmt.Println("└─────┴────────────────────────────────┴────────────────────────────────────────┘")
	fmt.Printf("\nTotal: %d pièce(s)\n", count)

	return nil
}

func printUsage() {
	fmt.Println(`recycle - Gestionnaire de pièces techniques

Usage:
  recycle <commande> [options]

Commandes:
  add    Ajouter une pièce au stock
  list   Lister toutes les pièces

Exemples:
  recycle add --name="Moteur Essuie-Glace" --props='{"volts":12, "axe":6}'
  recycle list`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	db, err := initDB()
	if err != nil {
		log.Fatalf("Erreur DB: %v", err)
	}
	defer db.Close()

	cmd := os.Args[1]

	switch cmd {
	case "add":
		if err := cmdAdd(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur add: %v", err)
		}
	case "list":
		if err := cmdList(db); err != nil {
			log.Fatalf("Erreur list: %v", err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Commande inconnue: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
