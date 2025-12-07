package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ImportStats contient les statistiques d'un import
type ImportStats struct {
	Total     int
	Imported  int
	Skipped   int
	Errors    int
	Duration  time.Duration
	ErrorMsgs []string
}

// ImportOptions contient les options d'import
type ImportOptions struct {
	FilePath   string
	TypeName   string // Type par défaut si non spécifié dans le fichier
	DryRun     bool   // Simuler sans écrire en DB
	StopOnErr  bool   // Arrêter au premier erreur
	Verbose    bool   // Afficher chaque ligne importée
}

// ImportFromFile importe des pièces depuis un fichier CSV ou JSON
func ImportFromFile(db *sql.DB, opts ImportOptions) (*ImportStats, error) {
	ext := strings.ToLower(filepath.Ext(opts.FilePath))

	switch ext {
	case ".csv":
		return importCSV(db, opts)
	case ".json":
		return importJSON(db, opts)
	default:
		return nil, fmt.Errorf("format non supporté: %s (utilisez .csv ou .json)", ext)
	}
}

// importCSV importe depuis un fichier CSV
// Format attendu: type,name,prop1,prop2,prop3...
// La première ligne doit être l'en-tête avec les noms des colonnes
func importCSV(db *sql.DB, opts ImportOptions) (*ImportStats, error) {
	file, err := os.Open(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Nombre de champs variable

	// Lire l'en-tête
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("erreur lecture en-tête: %v", err)
	}

	// Nettoyer les en-têtes
	for i, h := range headers {
		headers[i] = strings.TrimSpace(strings.ToLower(h))
	}

	// Trouver les indices des colonnes spéciales
	typeIdx := findIndex(headers, "type")
	nameIdx := findIndex(headers, "name", "nom")

	if nameIdx == -1 {
		return nil, fmt.Errorf("colonne 'name' ou 'nom' requise dans l'en-tête CSV")
	}

	stats := &ImportStats{}
	start := time.Now()

	// Commencer une transaction pour l'import en masse
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erreur transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO parts (type, name, props) VALUES (?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("erreur préparation: %v", err)
	}
	defer stmt.Close()

	lineNum := 1 // Ligne 1 = en-tête
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: %v", lineNum, err))
			if opts.StopOnErr {
				return stats, fmt.Errorf("erreur ligne %d: %v", lineNum, err)
			}
			continue
		}

		stats.Total++

		// Extraire le type
		typeName := opts.TypeName
		if typeIdx != -1 && typeIdx < len(record) {
			if t := strings.TrimSpace(record[typeIdx]); t != "" {
				typeName = t
			}
		}

		// Extraire le nom
		if nameIdx >= len(record) {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: colonne 'name' manquante", lineNum))
			continue
		}
		name := strings.TrimSpace(record[nameIdx])
		if name == "" {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: nom vide", lineNum))
			continue
		}

		// Construire les props à partir des autres colonnes
		props := make(map[string]interface{})
		for i, header := range headers {
			if i == typeIdx || i == nameIdx {
				continue // Ignorer type et name
			}
			if i >= len(record) {
				continue
			}
			value := strings.TrimSpace(record[i])
			if value == "" {
				continue
			}

			// Essayer de convertir en nombre
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				props[header] = num
			} else {
				props[header] = value
			}
		}

		// Normaliser les unités
		fieldUnits := GetFieldUnits(typeName)
		normalizedProps, err := NormalizeProps(props, fieldUnits)
		if err != nil {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: %v", lineNum, err))
			if opts.StopOnErr {
				return stats, fmt.Errorf("ligne %d: %v", lineNum, err)
			}
			continue
		}

		// Valider selon le template
		if typeName != "" {
			if err := ValidateProps(typeName, normalizedProps); err != nil {
				stats.Errors++
				stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: %v", lineNum, err))
				if opts.StopOnErr {
					return stats, fmt.Errorf("ligne %d: %v", lineNum, err)
				}
				continue
			}
		}

		// Sérialiser en JSON
		propsJSON, err := json.Marshal(normalizedProps)
		if err != nil {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: erreur JSON: %v", lineNum, err))
			continue
		}

		// Insérer en DB (sauf si dry-run)
		if !opts.DryRun {
			_, err = stmt.Exec(typeName, name, string(propsJSON))
			if err != nil {
				stats.Errors++
				stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("ligne %d: erreur DB: %v", lineNum, err))
				if opts.StopOnErr {
					return stats, fmt.Errorf("ligne %d: %v", lineNum, err)
				}
				continue
			}
		}

		stats.Imported++
		if opts.Verbose {
			fmt.Printf("  ✓ %s [%s] %s\n", typeName, name, string(propsJSON))
		}
	}

	// Commit la transaction
	if !opts.DryRun {
		if err := tx.Commit(); err != nil {
			return stats, fmt.Errorf("erreur commit: %v", err)
		}
	}

	stats.Duration = time.Since(start)
	return stats, nil
}

// importJSON importe depuis un fichier JSON
// Format attendu: tableau d'objets avec "type", "name", et autres propriétés
func importJSON(db *sql.DB, opts ImportOptions) (*ImportStats, error) {
	file, err := os.Open(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier: %v", err)
	}
	defer file.Close()

	var records []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("erreur parsing JSON: %v", err)
	}

	stats := &ImportStats{Total: len(records)}
	start := time.Now()

	// Commencer une transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erreur transaction: %v", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO parts (type, name, props) VALUES (?, ?, ?)")
	if err != nil {
		return nil, fmt.Errorf("erreur préparation: %v", err)
	}
	defer stmt.Close()

	for i, record := range records {
		lineNum := i + 1

		// Extraire le type
		typeName := opts.TypeName
		if t, ok := record["type"].(string); ok && t != "" {
			typeName = t
		}
		delete(record, "type")

		// Extraire le nom
		name := ""
		if n, ok := record["name"].(string); ok {
			name = n
		} else if n, ok := record["nom"].(string); ok {
			name = n
		}
		delete(record, "name")
		delete(record, "nom")

		if name == "" {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("enregistrement %d: nom manquant", lineNum))
			if opts.StopOnErr {
				return stats, fmt.Errorf("enregistrement %d: nom manquant", lineNum)
			}
			continue
		}

		// Les propriétés restantes sont les props
		props := record

		// Normaliser les unités
		fieldUnits := GetFieldUnits(typeName)
		normalizedProps, err := NormalizeProps(props, fieldUnits)
		if err != nil {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("enregistrement %d: %v", lineNum, err))
			if opts.StopOnErr {
				return stats, fmt.Errorf("enregistrement %d: %v", lineNum, err)
			}
			continue
		}

		// Valider selon le template
		if typeName != "" {
			if err := ValidateProps(typeName, normalizedProps); err != nil {
				stats.Errors++
				stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("enregistrement %d: %v", lineNum, err))
				if opts.StopOnErr {
					return stats, fmt.Errorf("enregistrement %d: %v", lineNum, err)
				}
				continue
			}
		}

		// Sérialiser en JSON
		propsJSON, err := json.Marshal(normalizedProps)
		if err != nil {
			stats.Errors++
			stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("enregistrement %d: erreur JSON: %v", lineNum, err))
			continue
		}

		// Insérer en DB
		if !opts.DryRun {
			_, err = stmt.Exec(typeName, name, string(propsJSON))
			if err != nil {
				stats.Errors++
				stats.ErrorMsgs = append(stats.ErrorMsgs, fmt.Sprintf("enregistrement %d: erreur DB: %v", lineNum, err))
				if opts.StopOnErr {
					return stats, fmt.Errorf("enregistrement %d: %v", lineNum, err)
				}
				continue
			}
		}

		stats.Imported++
		if opts.Verbose {
			fmt.Printf("  ✓ %s [%s] %s\n", typeName, name, string(propsJSON))
		}
	}

	// Commit
	if !opts.DryRun {
		if err := tx.Commit(); err != nil {
			return stats, fmt.Errorf("erreur commit: %v", err)
		}
	}

	stats.Duration = time.Since(start)
	return stats, nil
}

// findIndex trouve l'index d'une colonne par ses noms possibles
func findIndex(headers []string, names ...string) int {
	for i, h := range headers {
		for _, name := range names {
			if h == name {
				return i
			}
		}
	}
	return -1
}

// PrintImportStats affiche les statistiques d'import
func PrintImportStats(stats *ImportStats, dryRun bool) {
	if dryRun {
		fmt.Println("\n📋 Simulation (dry-run) - Aucune donnée écrite")
	}

	fmt.Println("\n┌─────────────────────────────────────┐")
	fmt.Println("│         Résumé de l'import          │")
	fmt.Println("├─────────────────────────────────────┤")
	fmt.Printf("│  Total lu:        %6d            │\n", stats.Total)
	fmt.Printf("│  Importé:         %6d ✓          │\n", stats.Imported)
	if stats.Errors > 0 {
		fmt.Printf("│  Erreurs:         %6d ✗          │\n", stats.Errors)
	}
	fmt.Printf("│  Durée:           %6.2fs           │\n", stats.Duration.Seconds())
	if stats.Total > 0 {
		rate := float64(stats.Imported) / stats.Duration.Seconds()
		fmt.Printf("│  Vitesse:         %6.0f/s          │\n", rate)
	}
	fmt.Println("└─────────────────────────────────────┘")

	// Afficher les 5 premières erreurs
	if len(stats.ErrorMsgs) > 0 {
		fmt.Println("\nPremières erreurs:")
		limit := 5
		if len(stats.ErrorMsgs) < limit {
			limit = len(stats.ErrorMsgs)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  ✗ %s\n", stats.ErrorMsgs[i])
		}
		if len(stats.ErrorMsgs) > 5 {
			fmt.Printf("  ... et %d autres erreurs\n", len(stats.ErrorMsgs)-5)
		}
	}
}

