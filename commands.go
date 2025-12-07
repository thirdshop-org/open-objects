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

	// Parser les props
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

	// Normaliser les unités
	fieldUnits := GetFieldUnits(*typeName)
	normalizedProps, err := NormalizeProps(propsMap, fieldUnits)
	if err != nil {
		return fmt.Errorf("erreur de normalisation: %v", err)
	}

	// Sérialiser les props normalisées
	normalizedJSON, err := json.Marshal(normalizedProps)
	if err != nil {
		return fmt.Errorf("erreur sérialisation: %v", err)
	}

	result, err := db.Exec("INSERT INTO parts (type, name, props) VALUES (?, ?, ?)", *typeName, *name, string(normalizedJSON))
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	fmt.Printf("✓ Pièce ajoutée [ID: %d]\n", id)
	if *typeName != "" {
		fmt.Printf("  Type: %s\n", *typeName)
	}
	fmt.Printf("  Nom: %s\n", *name)
	
	// Afficher les props normalisées avec indication des conversions
	if *props != string(normalizedJSON) {
		fmt.Printf("  Props (normalisées): %s\n", string(normalizedJSON))
		fmt.Printf("  Props (originales):  %s\n", *props)
	} else {
		fmt.Printf("  Props: %s\n", *props)
	}

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

	// Parser le critère de propriété si présent
	var propName, propExact string
	var propMin, propMax float64
	var isRange bool

	if *propSearch != "" {
		criteria, err := ParseSearchProp(*propSearch)
		if err != nil {
			return err
		}
		propName = criteria.PropName
		isRange = criteria.IsRange
		propExact = criteria.ExactVal
		propMin = criteria.MinVal
		propMax = criteria.MaxVal
	}

	// Requête unique avec CTEs pour lisibilité
	query := `
		WITH 
		-- Paramètres de recherche
		params AS (
			SELECT 
				$type      AS filter_type,
				$name      AS filter_name,
				$prop_name AS prop_name,
				$prop_exact AS prop_exact,
				$prop_min  AS prop_min,
				$prop_max  AS prop_max,
				$is_range  AS is_range
		),
		
		-- Filtre par type
		filtered_by_type AS (
			SELECT p.* 
			FROM parts p, params
			WHERE params.filter_type = '' 
			   OR p.type = params.filter_type
		),
		
		-- Filtre par nom
		filtered_by_name AS (
			SELECT f.* 
			FROM filtered_by_type f, params
			WHERE params.filter_name = '' 
			   OR f.name LIKE '%' || params.filter_name || '%'
		),
		
		-- Filtre par propriété JSON
		filtered_by_prop AS (
			SELECT f.* 
			FROM filtered_by_name f, params
			WHERE params.prop_name = ''
			   OR (
			       CASE 
			           WHEN params.is_range THEN
			               CAST(json_extract(f.props, '$.' || params.prop_name) AS REAL) 
			               BETWEEN params.prop_min AND params.prop_max
			           ELSE
			               CAST(json_extract(f.props, '$.' || params.prop_name) AS TEXT) = params.prop_exact
			       END
			   )
		)
		
		SELECT id, type, name, props 
		FROM filtered_by_prop
		ORDER BY id
	`

	rows, err := db.Query(query,
		sql.Named("type", *typeName),
		sql.Named("name", *nameSearch),
		sql.Named("prop_name", propName),
		sql.Named("prop_exact", propExact),
		sql.Named("prop_min", propMin),
		sql.Named("prop_max", propMax),
		sql.Named("is_range", isRange),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	return printPartsTable(rows, "Résultats")
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

func cmdImport(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	filePath := fs.String("file", "", "Chemin vers le fichier CSV ou JSON")
	typeName := fs.String("type", "", "Type par défaut pour les pièces (optionnel)")
	dryRun := fs.Bool("dry-run", false, "Simuler l'import sans écrire en base")
	stopOnErr := fs.Bool("stop-on-error", false, "Arrêter au premier erreur")
	verbose := fs.Bool("verbose", false, "Afficher chaque pièce importée")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" {
		return fmt.Errorf("le fichier est requis (--file=stock.csv)")
	}

	opts := ImportOptions{
		FilePath:  *filePath,
		TypeName:  *typeName,
		DryRun:    *dryRun,
		StopOnErr: *stopOnErr,
		Verbose:   *verbose,
	}

	fmt.Printf("📦 Import depuis: %s\n", *filePath)
	if *typeName != "" {
		fmt.Printf("   Type par défaut: %s\n", *typeName)
	}

	stats, err := ImportFromFile(db, opts)
	if err != nil {
		return err
	}

	PrintImportStats(stats, *dryRun)
	return nil
}

// --- Helpers d'affichage ---

func printPartsTable(rows *sql.Rows, countLabel string) error {
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
