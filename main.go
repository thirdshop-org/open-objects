package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

const dbPath = "recycle.db"
const templatesDir = "templates"

// Template représente un archétype de pièce
type Template struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Required    []string `yaml:"required"`
	Optional    []string `yaml:"optional"`
}

var templates = make(map[string]*Template)

func loadTemplates() error {
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Pas de templates, c'est OK
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(templatesDir, entry.Name()))
		if err != nil {
			return err
		}

		var tmpl Template
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return fmt.Errorf("erreur parsing %s: %v", entry.Name(), err)
		}

		templates[tmpl.Name] = &tmpl
	}

	return nil
}

func validateProps(typeName string, props map[string]interface{}) error {
	tmpl, exists := templates[typeName]
	if !exists {
		return nil // Type libre, pas de validation
	}

	for _, req := range tmpl.Required {
		if _, ok := props[req]; !ok {
			return fmt.Errorf("propriété requise manquante: %s (type %s)", req, typeName)
		}
	}

	return nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Migration: ajouter colonne type si elle n'existe pas
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS parts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT DEFAULT '',
			name TEXT NOT NULL,
			props JSON
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Vérifier si la colonne type existe, sinon l'ajouter
	rows, err := db.Query("PRAGMA table_info(parts)")
	if err != nil {
		db.Close()
		return nil, err
	}
	hasType := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		if name == "type" {
			hasType = true
		}
	}
	rows.Close()

	if !hasType {
		_, err = db.Exec("ALTER TABLE parts ADD COLUMN type TEXT DEFAULT ''")
		if err != nil {
			db.Close()
			return nil, err
		}
	}

	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_parts_name ON parts (name)")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_parts_type ON parts (type)")

	return db, nil
}

func cmdAdd(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	typeName := fs.String("type", "", "Type de pièce (ex: roulement, moteur)")
	name := fs.String("name", "", "Nom de la pièce")
	props := fs.String("props", "{}", "Propriétés JSON de la pièce")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("le nom est requis (--name)")
	}

	// Parser et valider les props
	var propsMap map[string]interface{}
	if err := json.Unmarshal([]byte(*props), &propsMap); err != nil {
		return fmt.Errorf("props invalide: %v", err)
	}

	// Valider selon le template si un type est spécifié
	if *typeName != "" {
		if err := validateProps(*typeName, propsMap); err != nil {
			return err
		}
	}

	result, err := db.Exec("INSERT INTO parts (type, name, props) VALUES (?, ?, ?)", *typeName, *name, *props)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	fmt.Printf("✓ Pièce ajoutée [ID: %d]\n", id)
	if *typeName != "" {
		fmt.Printf("  Type: %s\n", *typeName)
	}
	fmt.Printf("  Nom: %s\n", *name)
	fmt.Printf("  Props: %s\n", *props)

	return nil
}

func cmdList(db *sql.DB) error {
	rows, err := db.Query("SELECT id, type, name, props FROM parts ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	fmt.Println("┌─────┬──────────────┬────────────────────────────┬────────────────────────────────────────┐")
	fmt.Println("│ ID  │ Type         │ Nom                        │ Propriétés                             │")
	fmt.Println("├─────┼──────────────┼────────────────────────────┼────────────────────────────────────────┤")

	for rows.Next() {
		var id int
		var typeName, name string
		var props sql.NullString

		if err := rows.Scan(&id, &typeName, &name, &props); err != nil {
			return err
		}

		propsStr := "{}"
		if props.Valid {
			propsStr = props.String
		}

		// Tronquer si trop long
		if len(typeName) > 12 {
			typeName = typeName[:9] + "..."
		}
		if len(name) > 26 {
			name = name[:23] + "..."
		}
		if len(propsStr) > 38 {
			propsStr = propsStr[:35] + "..."
		}

		fmt.Printf("│ %-3d │ %-12s │ %-26s │ %-38s │\n", id, typeName, name, propsStr)
		count++
	}

	fmt.Println("└─────┴──────────────┴────────────────────────────┴────────────────────────────────────────┘")
	fmt.Printf("\nTotal: %d pièce(s)\n", count)

	return nil
}

// SearchCriteria représente un critère de recherche
type SearchCriteria struct {
	PropName string
	IsRange  bool
	ExactVal string
	MinVal   float64
	MaxVal   float64
}

// parseSearchProp parse une prop comme "d_int:10..10.5" ou "type:billes"
func parseSearchProp(prop string) (*SearchCriteria, error) {
	parts := strings.SplitN(prop, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("format invalide: %s (attendu: prop:valeur ou prop:min..max)", prop)
	}

	criteria := &SearchCriteria{PropName: parts[0]}
	value := parts[1]

	// Vérifier si c'est un range (contient ..)
	rangeRegex := regexp.MustCompile(`^([\d.]+)\.\.([\d.]+)$`)
	if matches := rangeRegex.FindStringSubmatch(value); matches != nil {
		min, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return nil, fmt.Errorf("valeur min invalide: %s", matches[1])
		}
		max, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return nil, fmt.Errorf("valeur max invalide: %s", matches[2])
		}
		criteria.IsRange = true
		criteria.MinVal = min
		criteria.MaxVal = max
	} else {
		criteria.IsRange = false
		criteria.ExactVal = value
	}

	return criteria, nil
}

func cmdSearch(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	typeName := fs.String("type", "", "Filtrer par type de pièce")
	propSearch := fs.String("prop", "", "Recherche par propriété (ex: d_int:10 ou d_int:10..10.5)")
	nameSearch := fs.String("name", "", "Recherche par nom (partiel)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Construire la requête
	query := `
		SELECT 
			id, 
			type, 
			name, 
			props 
		FROM 
			parts 
		WHERE 
			CASE 
				WHEN $type IS NOT NULL AND $type != '' 
				THEN type = $type 
				ELSE FALSE 
			END
			OR
			CASE 
				WHEN $name IS NOT NULL 
				AND $name != '' 
				THEN name LIKE '%' || $name || '%'
				ELSE FALSE
			END
	`
	rows, err := db.Query(query, sql.Named("type", typeName), sql.Named("name", *nameSearch))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Parser le critère de recherche par propriété
	var criteria *SearchCriteria
	if *propSearch != "" {
		criteria, err = parseSearchProp(*propSearch)
		if err != nil {
			return err
		}
	}

	count := 0
	fmt.Println("┌─────┬──────────────┬────────────────────────────┬────────────────────────────────────────┐")
	fmt.Println("│ ID  │ Type         │ Nom                        │ Propriétés                             │")
	fmt.Println("├─────┼──────────────┼────────────────────────────┼────────────────────────────────────────┤")

	for rows.Next() {
		var id int
		var typeName, name string
		var propsRaw sql.NullString

		if err := rows.Scan(&id, &typeName, &name, &propsRaw); err != nil {
			return err
		}

		propsStr := "{}"
		if propsRaw.Valid {
			propsStr = propsRaw.String
		}

		// Filtrage par propriété (en mémoire pour flexibilité JSON)
		if criteria != nil {
			var propsMap map[string]interface{}
			if err := json.Unmarshal([]byte(propsStr), &propsMap); err != nil {
				continue
			}

			propVal, exists := propsMap[criteria.PropName]
			if !exists {
				continue
			}

			if criteria.IsRange {
				// Convertir en float pour comparaison
				var numVal float64
				switch v := propVal.(type) {
				case float64:
					numVal = v
				case int:
					numVal = float64(v)
				case string:
					numVal, err = strconv.ParseFloat(v, 64)
					if err != nil {
						continue
					}
				default:
					continue
				}

				if numVal < criteria.MinVal || numVal > criteria.MaxVal {
					continue
				}
			} else {
				// Recherche exacte
				propStr := fmt.Sprintf("%v", propVal)
				if propStr != criteria.ExactVal {
					continue
				}
			}
		}

		// Tronquer si trop long
		displayType := typeName
		displayName := name
		displayProps := propsStr
		if len(displayType) > 12 {
			displayType = displayType[:9] + "..."
		}
		if len(displayName) > 26 {
			displayName = displayName[:23] + "..."
		}
		if len(displayProps) > 38 {
			displayProps = displayProps[:35] + "..."
		}

		fmt.Printf("│ %-3d │ %-12s │ %-26s │ %-38s │\n", id, displayType, displayName, displayProps)
		count++
	}

	fmt.Println("└─────┴──────────────┴────────────────────────────┴────────────────────────────────────────┘")
	fmt.Printf("\nRésultats: %d pièce(s)\n", count)

	return nil
}

func cmdTemplates() error {
	if len(templates) == 0 {
		fmt.Println("Aucun template trouvé dans", templatesDir)
		return nil
	}

	fmt.Println("Templates disponibles:")
	fmt.Println()

	for name, tmpl := range templates {
		fmt.Printf("▸ %s\n", name)
		fmt.Printf("  %s\n", tmpl.Description)
		fmt.Printf("  Requis: %s\n", strings.Join(tmpl.Required, ", "))
		if len(tmpl.Optional) > 0 {
			fmt.Printf("  Optionnel: %s\n", strings.Join(tmpl.Optional, ", "))
		}
		fmt.Println()
	}

	return nil
}

func printUsage() {
	fmt.Println(`recycle - Gestionnaire de pièces techniques

Usage:
  recycle <commande> [options]

Commandes:
  add        Ajouter une pièce au stock
  list       Lister toutes les pièces
  search     Rechercher des pièces
  templates  Afficher les types de pièces disponibles

Exemples:
  recycle add --type=moteur --name="Moteur Essuie-Glace" --props='{"volts":12, "watts":50}'
  recycle add --type=roulement --name="SKF 6204" --props='{"d_int":20, "d_ext":47, "largeur":14}'
  recycle list
  recycle search --type=roulement
  recycle search --type=roulement --prop="d_int:10..25"
  recycle search --name="SKF"
  recycle templates`)
}

func main() {
	// Charger les templates
	if err := loadTemplates(); err != nil {
		log.Printf("Warning: impossible de charger les templates: %v", err)
	}

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
	case "search":
		if err := cmdSearch(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur search: %v", err)
		}
	case "templates":
		if err := cmdTemplates(); err != nil {
			log.Fatalf("Erreur templates: %v", err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Commande inconnue: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}
