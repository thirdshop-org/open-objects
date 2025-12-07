package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
)

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
		if err := ValidateProps(*typeName, propsMap); err != nil {
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

	return printPartsTable(rows, "Total")
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
		SELECT id, type, name, props 
		FROM parts 
		WHERE 
			CASE 
				WHEN $type IS NOT NULL AND $type != '' 
				THEN type = $type 
				ELSE FALSE 
			END
			OR
			CASE 
				WHEN $name IS NOT NULL AND $name != '' 
				THEN name LIKE '%' || $name || '%'
				ELSE FALSE
			END
	`
	rows, err := db.Query(query, sql.Named("type", *typeName), sql.Named("name", *nameSearch))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Parser le critère de recherche par propriété
	var criteria *SearchCriteria
	if *propSearch != "" {
		criteria, err = ParseSearchProp(*propSearch)
		if err != nil {
			return err
		}
	}

	return printPartsTableFiltered(rows, criteria, "Résultats")
}

func cmdTemplates() error {
	if len(Templates) == 0 {
		fmt.Println("Aucun template trouvé dans", templatesDir)
		return nil
	}

	fmt.Println("Templates disponibles:")
	fmt.Println()

	for name, tmpl := range Templates {
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

// --- Helpers d'affichage ---

func printPartsTable(rows *sql.Rows, countLabel string) error {
	return printPartsTableFiltered(rows, nil, countLabel)
}

func printPartsTableFiltered(rows *sql.Rows, criteria *SearchCriteria, countLabel string) error {
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

		// Filtrage par propriété si critère spécifié
		if criteria != nil {
			var propsMap map[string]interface{}
			if err := json.Unmarshal([]byte(propsStr), &propsMap); err != nil {
				continue
			}

			propVal, exists := propsMap[criteria.PropName]
			if !exists || !criteria.MatchesCriteria(propVal) {
				continue
			}
		}

		// Tronquer si trop long
		displayType := truncate(typeName, 12)
		displayName := truncate(name, 26)
		displayProps := truncate(propsStr, 38)

		fmt.Printf("│ %-3d │ %-12s │ %-26s │ %-38s │\n", id, displayType, displayName, displayProps)
		count++
	}

	fmt.Println("└─────┴──────────────┴────────────────────────────┴────────────────────────────────────────┘")
	fmt.Printf("\n%s: %d pièce(s)\n", countLabel, count)

	return nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
