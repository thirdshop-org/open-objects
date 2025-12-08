package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupData représente la structure complète d'un backup
type BackupData struct {
	Version     string            `json:"version"`
	GeneratedAt string            `json:"generated_at"`
	Locations   []BackupLocation  `json:"locations"`
	Parts       []BackupPart      `json:"parts"`
	Attachments []BackupAttachment `json:"attachments"`
}

// BackupLocation représente une localisation dans le backup
type BackupLocation struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ParentID    *int   `json:"parent_id,omitempty"`
	LocType     string `json:"loc_type"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// BackupPart représente une pièce dans le backup
type BackupPart struct {
	ID         int                    `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Props      map[string]interface{} `json:"props"`
	LocationID *int                   `json:"location_id,omitempty"`
	CreatedAt  string                 `json:"created_at"`
}

// BackupAttachment représente un fichier attaché dans le backup
type BackupAttachment struct {
	ID        int    `json:"id"`
	PartID    int    `json:"part_id"`
	Filename  string `json:"filename"`
	Filepath  string `json:"filepath"`
	Filetype  string `json:"filetype"`
	Filesize  int64  `json:"filesize"`
	CreatedAt string `json:"created_at"`
}

// CreateBackup crée un fichier de sauvegarde complet
func CreateBackup(db *sql.DB, filename string) error {
	fmt.Printf("📦 Création de la sauvegarde: %s\n", filename)

	// Créer les données de backup
	backup := BackupData{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Exporter les localisations
	if err := exportLocations(db, &backup); err != nil {
		return fmt.Errorf("erreur export locations: %v", err)
	}

	// Exporter les pièces
	if err := exportParts(db, &backup); err != nil {
		return fmt.Errorf("erreur export parts: %v", err)
	}

	// Exporter les attachments
	if err := exportAttachments(db, &backup); err != nil {
		return fmt.Errorf("erreur export attachments: %v", err)
	}

	// Écrire le fichier JSON
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("impossible de créer le fichier: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backup); err != nil {
		return fmt.Errorf("erreur écriture JSON: %v", err)
	}

	fmt.Printf("✓ Sauvegarde créée avec succès\n")
	fmt.Printf("  📍 Localisations: %d\n", len(backup.Locations))
	fmt.Printf("  🔧 Pièces: %d\n", len(backup.Parts))
	fmt.Printf("  📎 Fichiers: %d\n", len(backup.Attachments))
	fmt.Printf("  📁 Taille: %.1f KB\n", float64(getFileSize(filename))/1024)

	return nil
}

// RestoreFromBackup restaure la base depuis un fichier de sauvegarde
func RestoreFromBackup(db *sql.DB, filename string) error {
	fmt.Printf("🔄 Restauration depuis: %s\n", filename)

	// Lire le fichier de backup
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir le fichier: %v", err)
	}
	defer file.Close()

	var backup BackupData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&backup); err != nil {
		return fmt.Errorf("erreur lecture JSON: %v", err)
	}

	fmt.Printf("📊 Sauvegarde détectée: v%s (%s)\n", backup.Version, backup.GeneratedAt[:19])
	fmt.Printf("  📍 Localisations: %d\n", len(backup.Locations))
	fmt.Printf("  🔧 Pièces: %d\n", len(backup.Parts))
	fmt.Printf("  📎 Fichiers: %d\n", len(backup.Attachments))

	// Commencer une transaction pour la restauration
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("erreur transaction: %v", err)
	}
	defer tx.Rollback()

	// Nettoyer les tables existantes
	if err := cleanTables(tx); err != nil {
		return fmt.Errorf("erreur nettoyage: %v", err)
	}

	// Restaurer les localisations
	if err := restoreLocations(tx, backup.Locations); err != nil {
		return fmt.Errorf("erreur restauration locations: %v", err)
	}

	// Restaurer les pièces
	if err := restoreParts(tx, backup.Parts); err != nil {
		return fmt.Errorf("erreur restauration parts: %v", err)
	}

	// Restaurer les attachments
	if err := restoreAttachments(tx, backup.Attachments); err != nil {
		return fmt.Errorf("erreur restauration attachments: %v", err)
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erreur commit: %v", err)
	}

	fmt.Printf("✓ Restauration terminée avec succès\n")
	return nil
}

// exportLocations exporte toutes les localisations
func exportLocations(db *sql.DB, backup *BackupData) error {
	rows, err := db.Query(`
		SELECT id, name, parent_id, loc_type, description, created_at
		FROM locations
		ORDER BY id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var loc BackupLocation
		var parentID sql.NullInt64
		var description sql.NullString
		var createdAt string

		if err := rows.Scan(&loc.ID, &loc.Name, &parentID, &loc.LocType, &description, &createdAt); err != nil {
			return err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			loc.ParentID = &pid
		}

		if description.Valid {
			loc.Description = description.String
		}

		loc.CreatedAt = createdAt
		backup.Locations = append(backup.Locations, loc)
	}

	return nil
}

// exportParts exporte toutes les pièces
func exportParts(db *sql.DB, backup *BackupData) error {
	rows, err := db.Query(`
		SELECT p.id, p.type, p.name, p.props, p.location_id,
			   COALESCE(strftime('%Y-%m-%dT%H:%M:%fZ', p.rowid, 'unixepoch'), 'unknown') as created_at
		FROM parts p
		ORDER BY p.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var part BackupPart
		var propsJSON string
		var locationID sql.NullInt64
		var createdAt string

		if err := rows.Scan(&part.ID, &part.Type, &part.Name, &propsJSON, &locationID, &createdAt); err != nil {
			return err
		}

		// Parser les propriétés JSON
		if err := json.Unmarshal([]byte(propsJSON), &part.Props); err != nil {
			return fmt.Errorf("erreur parsing props ID %d: %v", part.ID, err)
		}

		if locationID.Valid {
			lid := int(locationID.Int64)
			part.LocationID = &lid
		}

		part.CreatedAt = createdAt
		backup.Parts = append(backup.Parts, part)
	}

	return nil
}

// exportAttachments exporte tous les fichiers attachés
func exportAttachments(db *sql.DB, backup *BackupData) error {
	rows, err := db.Query(`
		SELECT id, part_id, filename, filepath, filetype, filesize, created_at
		FROM attachments
		ORDER BY id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var att BackupAttachment

		if err := rows.Scan(&att.ID, &att.PartID, &att.Filename, &att.Filepath, &att.Filetype, &att.Filesize, &att.CreatedAt); err != nil {
			return err
		}

		backup.Attachments = append(backup.Attachments, att)
	}

	return nil
}

// cleanTables nettoie toutes les tables avant la restauration
func cleanTables(tx *sql.Tx) error {
	tables := []string{"attachments", "parts", "locations"}

	for _, table := range tables {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("erreur nettoyage %s: %v", table, err)
		}
	}

	return nil
}

// restoreLocations restaure les localisations
func restoreLocations(tx *sql.Tx, locations []BackupLocation) error {
	for _, loc := range locations {
		var parentID interface{}
		if loc.ParentID != nil {
			parentID = *loc.ParentID
		} else {
			parentID = nil
		}

		_, err := tx.Exec(`
			INSERT INTO locations (id, name, parent_id, loc_type, description, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, loc.ID, loc.Name, parentID, loc.LocType, loc.Description, loc.CreatedAt)

		if err != nil {
			return fmt.Errorf("erreur restauration location %d: %v", loc.ID, err)
		}
	}

	return nil
}

// restoreParts restaure les pièces
func restoreParts(tx *sql.Tx, parts []BackupPart) error {
	for _, part := range parts {
		// Sérialiser les propriétés
		propsJSON, err := json.Marshal(part.Props)
		if err != nil {
			return fmt.Errorf("erreur sérialisation props ID %d: %v", part.ID, err)
		}

		var locationID interface{}
		if part.LocationID != nil {
			locationID = *part.LocationID
		} else {
			locationID = nil
		}

		_, err = tx.Exec(`
			INSERT INTO parts (id, type, name, props, location_id)
			VALUES (?, ?, ?, ?, ?)
		`, part.ID, part.Type, part.Name, string(propsJSON), locationID)

		if err != nil {
			return fmt.Errorf("erreur restauration pièce %d: %v", part.ID, err)
		}
	}

	return nil
}

// restoreAttachments restaure les fichiers attachés
func restoreAttachments(tx *sql.Tx, attachments []BackupAttachment) error {
	for _, att := range attachments {
		_, err := tx.Exec(`
			INSERT INTO attachments (id, part_id, filename, filepath, filetype, filesize, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, att.ID, att.PartID, att.Filename, att.Filepath, att.Filetype, att.Filesize, att.CreatedAt)

		if err != nil {
			return fmt.Errorf("erreur restauration attachment %d: %v", att.ID, err)
		}
	}

	return nil
}

// getFileSize retourne la taille d'un fichier
func getFileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}

// ValidateBackupFile vérifie qu'un fichier de backup est valide
func ValidateBackupFile(filename string) (*BackupData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir le fichier: %v", err)
	}
	defer file.Close()

	var backup BackupData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&backup); err != nil {
		return nil, fmt.Errorf("fichier JSON invalide: %v", err)
	}

	// Validation basique
	if backup.Version == "" {
		return nil, fmt.Errorf("version manquante dans le backup")
	}

	if len(backup.Locations) == 0 && len(backup.Parts) == 0 {
		return nil, fmt.Errorf("backup vide (pas de localisations ni de pièces)")
	}

	return &backup, nil
}

// ListBackups liste les fichiers de backup disponibles
func ListBackups(directory string) ([]os.FileInfo, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	var backups []os.FileInfo
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			info, err := file.Info()
			if err != nil {
				continue
			}
			backups = append(backups, info)
		}
	}

	return backups, nil
}

