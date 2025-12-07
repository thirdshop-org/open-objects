package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const assetsDir = "assets"

// Attachment représente un fichier attaché à une pièce
type Attachment struct {
	ID        int
	PartID    int
	Filename  string
	Filepath  string
	Filetype  string
	Filesize  int64
	CreatedAt string
}

// FileTypeInfo contient les informations sur un type de fichier
type FileTypeInfo struct {
	Category string // "document", "image", "other"
	Icon     string
}

// KnownFileTypes mappe les extensions aux types de fichiers
var KnownFileTypes = map[string]FileTypeInfo{
	".pdf":  {"document", "📄"},
	".doc":  {"document", "📄"},
	".docx": {"document", "📄"},
	".txt":  {"document", "📄"},
	".odt":  {"document", "📄"},
	".jpg":  {"image", "🖼️"},
	".jpeg": {"image", "🖼️"},
	".png":  {"image", "🖼️"},
	".gif":  {"image", "🖼️"},
	".webp": {"image", "🖼️"},
	".bmp":  {"image", "🖼️"},
	".svg":  {"image", "🖼️"},
	".zip":  {"archive", "📦"},
	".rar":  {"archive", "📦"},
	".7z":   {"archive", "📦"},
	".stl":  {"3d", "🔧"},
	".step": {"3d", "🔧"},
	".stp":  {"3d", "🔧"},
	".dwg":  {"cad", "📐"},
	".dxf":  {"cad", "📐"},
}

// EnsureAssetsDir crée le dossier assets s'il n'existe pas
func EnsureAssetsDir() error {
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		return os.MkdirAll(assetsDir, 0755)
	}
	return nil
}

// AttachFile attache un fichier à une pièce
func AttachFile(db *sql.DB, partID int, sourcePath string) (*Attachment, error) {
	// Vérifier que la pièce existe
	var partName string
	err := db.QueryRow("SELECT name FROM parts WHERE id = ?", partID).Scan(&partName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pièce ID %d introuvable", partID)
	}
	if err != nil {
		return nil, fmt.Errorf("erreur DB: %v", err)
	}

	// Vérifier que le fichier source existe
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("fichier introuvable: %s", sourcePath)
	}
	if sourceInfo.IsDir() {
		return nil, fmt.Errorf("impossible d'attacher un dossier")
	}

	// Créer le dossier assets
	if err := EnsureAssetsDir(); err != nil {
		return nil, fmt.Errorf("impossible de créer le dossier assets: %v", err)
	}

	// Construire le nom du fichier destination
	originalName := filepath.Base(sourcePath)
	ext := strings.ToLower(filepath.Ext(originalName))
	baseName := strings.TrimSuffix(originalName, filepath.Ext(originalName))
	
	// Nettoyer le nom de base (remplacer les espaces, caractères spéciaux)
	baseName = sanitizeFilename(baseName)
	
	// Format: <part_id>_<nom_fichier>.<ext>
	destName := fmt.Sprintf("%d_%s%s", partID, baseName, ext)
	destPath := filepath.Join(assetsDir, destName)

	// Vérifier si un fichier avec le même nom existe déjà
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		destName = fmt.Sprintf("%d_%s_%d%s", partID, baseName, counter, ext)
		destPath = filepath.Join(assetsDir, destName)
		counter++
	}

	// Copier le fichier
	if err := copyFile(sourcePath, destPath); err != nil {
		return nil, fmt.Errorf("erreur de copie: %v", err)
	}

	// Déterminer le type de fichier
	fileType := "other"
	if info, ok := KnownFileTypes[ext]; ok {
		fileType = info.Category
	}

	// Enregistrer en base de données
	result, err := db.Exec(`
		INSERT INTO attachments (part_id, filename, filepath, filetype, filesize)
		VALUES (?, ?, ?, ?, ?)
	`, partID, originalName, destPath, fileType, sourceInfo.Size())
	if err != nil {
		// Nettoyer le fichier copié en cas d'erreur
		os.Remove(destPath)
		return nil, fmt.Errorf("erreur DB: %v", err)
	}

	attachID, _ := result.LastInsertId()

	return &Attachment{
		ID:       int(attachID),
		PartID:   partID,
		Filename: originalName,
		Filepath: destPath,
		Filetype: fileType,
		Filesize: sourceInfo.Size(),
	}, nil
}

// GetAttachments retourne tous les fichiers attachés à une pièce
func GetAttachments(db *sql.DB, partID int) ([]Attachment, error) {
	rows, err := db.Query(`
		SELECT id, part_id, filename, filepath, filetype, filesize, created_at
		FROM attachments
		WHERE part_id = ?
		ORDER BY created_at DESC
	`, partID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.PartID, &a.Filename, &a.Filepath, &a.Filetype, &a.Filesize, &a.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}

	return attachments, nil
}

// GetAttachmentCount retourne le nombre de fichiers attachés à une pièce
func GetAttachmentCount(db *sql.DB, partID int) int {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM attachments WHERE part_id = ?", partID).Scan(&count)
	return count
}

// GetAttachmentsForParts retourne un map des attachments par part_id
func GetAttachmentsForParts(db *sql.DB, partIDs []int) (map[int][]Attachment, error) {
	if len(partIDs) == 0 {
		return make(map[int][]Attachment), nil
	}

	// Construire la requête avec IN clause
	placeholders := make([]string, len(partIDs))
	args := make([]interface{}, len(partIDs))
	for i, id := range partIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, part_id, filename, filepath, filetype, filesize, created_at
		FROM attachments
		WHERE part_id IN (%s)
		ORDER BY part_id, created_at DESC
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]Attachment)
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.PartID, &a.Filename, &a.Filepath, &a.Filetype, &a.Filesize, &a.CreatedAt); err != nil {
			return nil, err
		}
		result[a.PartID] = append(result[a.PartID], a)
	}

	return result, nil
}

// DeleteAttachment supprime un fichier attaché
func DeleteAttachment(db *sql.DB, attachID int) error {
	// Récupérer le chemin du fichier
	var filepath string
	err := db.QueryRow("SELECT filepath FROM attachments WHERE id = ?", attachID).Scan(&filepath)
	if err == sql.ErrNoRows {
		return fmt.Errorf("attachement ID %d introuvable", attachID)
	}
	if err != nil {
		return err
	}

	// Supprimer le fichier
	os.Remove(filepath) // Ignorer l'erreur si le fichier n'existe plus

	// Supprimer de la base
	_, err = db.Exec("DELETE FROM attachments WHERE id = ?", attachID)
	return err
}

// ListPartAttachments affiche les fichiers attachés à une pièce
func ListPartAttachments(db *sql.DB, partID int) error {
	// Vérifier que la pièce existe
	var partName string
	err := db.QueryRow("SELECT name FROM parts WHERE id = ?", partID).Scan(&partName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("pièce ID %d introuvable", partID)
	}
	if err != nil {
		return err
	}

	attachments, err := GetAttachments(db, partID)
	if err != nil {
		return err
	}

	fmt.Printf("\n📎 Fichiers attachés à: %s (ID: %d)\n", partName, partID)
	fmt.Println(strings.Repeat("─", 60))

	if len(attachments) == 0 {
		fmt.Println("  Aucun fichier attaché")
		return nil
	}

	for _, a := range attachments {
		icon := "📁"
		ext := strings.ToLower(filepath.Ext(a.Filename))
		if info, ok := KnownFileTypes[ext]; ok {
			icon = info.Icon
		}

		sizeStr := formatFileSize(a.Filesize)
		fmt.Printf("  %s %s (%s)\n", icon, a.Filename, sizeStr)
		fmt.Printf("     → %s\n", a.Filepath)
	}

	fmt.Println()
	return nil
}

// --- Helpers ---

// copyFile copie un fichier source vers destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// sanitizeFilename nettoie un nom de fichier
func sanitizeFilename(name string) string {
	// Remplacer les caractères problématiques
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// formatFileSize formate une taille de fichier en format lisible
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// FormatAttachmentsSummary formate un résumé des attachments pour l'affichage
func FormatAttachmentsSummary(attachments []Attachment) string {
	if len(attachments) == 0 {
		return ""
	}

	// Compter par type
	docs := 0
	images := 0
	others := 0

	for _, a := range attachments {
		switch a.Filetype {
		case "document":
			docs++
		case "image":
			images++
		default:
			others++
		}
	}

	parts := []string{}
	if docs > 0 {
		parts = append(parts, fmt.Sprintf("📄%d", docs))
	}
	if images > 0 {
		parts = append(parts, fmt.Sprintf("🖼️%d", images))
	}
	if others > 0 {
		parts = append(parts, fmt.Sprintf("📁%d", others))
	}

	return strings.Join(parts, " ")
}

