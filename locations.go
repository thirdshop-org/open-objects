package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// LocationType définit le type de localisation
type LocationType string

const (
	LocTypeZone      LocationType = "ZONE"      // Atelier, Pièce
	LocTypeFurniture LocationType = "FURNITURE" // Armoire, Établi, Étagère
	LocTypeShelf     LocationType = "SHELF"     // Étagère, Rayon
	LocTypeBox       LocationType = "BOX"       // Boîte, Bac, Tiroir
)

// Location représente un emplacement dans l'arborescence
type Location struct {
	ID          int
	Name        string
	ParentID    sql.NullInt64
	LocType     string
	Description string
	CreatedAt   string
}

// LocationTypeIcons mappe les types vers des icônes
var LocationTypeIcons = map[string]string{
	"ZONE":      "🏭",
	"FURNITURE": "🗄️",
	"SHELF":     "📚",
	"BOX":       "📦",
}

// GetLocationIcon retourne l'icône pour un type de localisation
func GetLocationIcon(locType string) string {
	if icon, ok := LocationTypeIcons[locType]; ok {
		return icon
	}
	return "📍"
}

// CreateLocation crée une nouvelle localisation
func CreateLocation(db *sql.DB, name string, parentID *int, locType string, description string) (*Location, error) {
	// Valider le type
	if locType == "" {
		locType = "BOX"
	}
	locType = strings.ToUpper(locType)

	// Vérifier que le parent existe si spécifié
	var parentIDValue sql.NullInt64
	if parentID != nil {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM locations WHERE id = ?", *parentID).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("localisation parent ID %d introuvable", *parentID)
		}
		parentIDValue = sql.NullInt64{Int64: int64(*parentID), Valid: true}
	}

	// Insérer la localisation
	result, err := db.Exec(`
		INSERT INTO locations (name, parent_id, loc_type, description)
		VALUES (?, ?, ?, ?)
	`, name, parentIDValue, locType, description)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	return &Location{
		ID:          int(id),
		Name:        name,
		ParentID:    parentIDValue,
		LocType:     locType,
		Description: description,
	}, nil
}

// FindLocationByName cherche une localisation par son nom (insensible à la casse)
func FindLocationByName(db *sql.DB, name string) (*Location, error) {
	var loc Location
	var parentID sql.NullInt64

	err := db.QueryRow(`
		SELECT id, name, parent_id, loc_type, description
		FROM locations
		WHERE LOWER(name) = LOWER(?)
	`, name).Scan(&loc.ID, &loc.Name, &parentID, &loc.LocType, &loc.Description)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("localisation '%s' introuvable", name)
	}
	if err != nil {
		return nil, err
	}

	loc.ParentID = parentID
	return &loc, nil
}

// FindLocationByID cherche une localisation par son ID
func FindLocationByID(db *sql.DB, id int) (*Location, error) {
	var loc Location
	var parentID sql.NullInt64

	err := db.QueryRow(`
		SELECT id, name, parent_id, loc_type, description
		FROM locations
		WHERE id = ?
	`, id).Scan(&loc.ID, &loc.Name, &parentID, &loc.LocType, &loc.Description)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("localisation ID %d introuvable", id)
	}
	if err != nil {
		return nil, err
	}

	loc.ParentID = parentID
	return &loc, nil
}

// GetFullPath retourne le chemin complet d'une localisation (Atelier > Meuble > Boîte)
func GetFullPath(db *sql.DB, locationID int) (string, error) {
	var parts []string

	query := `
		WITH RECURSIVE location_tree AS (
			SELECT 
				id, 
				name, 
				parent_id, 
				0 as level 
			FROM 
				locations 
			WHERE
				id = ?
			UNION ALL
			SELECT 
				l.id, 
				l.name, 
				l.parent_id, 
				lt.level + 1 
			FROM 
				locations l
			JOIN 
				location_tree lt 
			ON 
				l.parent_id = lt.id 
				AND 
				l.id != lt.id 
				AND 
				lt.level < 100
		)
		SELECT 
			name, 
			level 
		FROM 
			location_tree 
		ORDER BY level DESC
	`
	rows, err := db.Query(query, locationID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var level int
		err := rows.Scan(&name, &level)
		if err != nil {
			return "", err
		}
		parts = append([]string{name}, parts...)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("localisation ID %d introuvable", locationID)
	}

	return strings.Join(parts, " > "), nil
}

// GetLocationWithPath retourne une localisation avec son chemin complet
func GetLocationWithPath(db *sql.DB, locationID int) (string, error) {
	loc, err := FindLocationByID(db, locationID)
	if err != nil {
		return "", err
	}

	path, err := GetFullPath(db, locationID)
	if err != nil {
		return "", err
	}

	icon := GetLocationIcon(loc.LocType)
	return fmt.Sprintf("%s %s (ID #%d)", icon, path, locationID), nil
}

// ListLocations liste toutes les localisations
func ListLocations(db *sql.DB) ([]Location, error) {
	rows, err := db.Query(`
		SELECT id, name, parent_id, loc_type, description
		FROM locations
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var loc Location
		var parentID sql.NullInt64
		if err := rows.Scan(&loc.ID, &loc.Name, &parentID, &loc.LocType, &loc.Description); err != nil {
			return nil, err
		}
		loc.ParentID = parentID
		locations = append(locations, loc)
	}

	return locations, nil
}

// ListRootLocations liste les localisations racines (sans parent)
func ListRootLocations(db *sql.DB) ([]Location, error) {
	rows, err := db.Query(`
		SELECT id, name, parent_id, loc_type, description
		FROM locations
		WHERE parent_id IS NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var loc Location
		var parentID sql.NullInt64
		if err := rows.Scan(&loc.ID, &loc.Name, &parentID, &loc.LocType, &loc.Description); err != nil {
			return nil, err
		}
		loc.ParentID = parentID
		locations = append(locations, loc)
	}

	return locations, nil
}

// ListChildLocations liste les enfants d'une localisation
func ListChildLocations(db *sql.DB, parentID int) ([]Location, error) {
	rows, err := db.Query(`
		SELECT id, name, parent_id, loc_type, description
		FROM locations
		WHERE parent_id = ?
		ORDER BY name
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var loc Location
		var pid sql.NullInt64
		if err := rows.Scan(&loc.ID, &loc.Name, &pid, &loc.LocType, &loc.Description); err != nil {
			return nil, err
		}
		loc.ParentID = pid
		locations = append(locations, loc)
	}

	return locations, nil
}

// MoveLocation déplace une localisation vers un nouveau parent
func MoveLocation(db *sql.DB, locationID int, newParentID *int) error {
	// Vérifier que la localisation existe
	_, err := FindLocationByID(db, locationID)
	if err != nil {
		return err
	}

	// Vérifier que le nouveau parent existe (si spécifié)
	if newParentID != nil {
		_, err := FindLocationByID(db, *newParentID)
		if err != nil {
			return fmt.Errorf("nouveau parent: %v", err)
		}

		// Vérifier qu'on ne crée pas de cycle (le nouveau parent ne doit pas être un descendant)
		if isDescendant(db, *newParentID, locationID) {
			return fmt.Errorf("impossible de déplacer vers un descendant (créerait un cycle)")
		}
	}

	var parentIDValue interface{}
	if newParentID != nil {
		parentIDValue = *newParentID
	} else {
		parentIDValue = nil
	}

	_, err = db.Exec("UPDATE locations SET parent_id = ? WHERE id = ?", parentIDValue, locationID)
	return err
}

// isDescendant vérifie si potentialDescendant est un descendant de ancestorID
func isDescendant(db *sql.DB, potentialDescendant int, ancestorID int) bool {
	currentID := potentialDescendant
	visited := make(map[int]bool)

	for {
		if currentID == ancestorID {
			return true
		}
		if visited[currentID] {
			return false // Cycle détecté, arrêter
		}
		visited[currentID] = true

		var parentID sql.NullInt64
		err := db.QueryRow("SELECT parent_id FROM locations WHERE id = ?", currentID).Scan(&parentID)
		if err != nil || !parentID.Valid {
			return false
		}
		currentID = int(parentID.Int64)
	}
}

// DeleteLocation supprime une localisation
func DeleteLocation(db *sql.DB, locationID int) error {
	// Vérifier qu'il n'y a pas de pièces liées
	var partCount int
	db.QueryRow("SELECT COUNT(*) FROM parts WHERE location_id = ?", locationID).Scan(&partCount)
	if partCount > 0 {
		return fmt.Errorf("impossible de supprimer: %d pièce(s) sont dans cette localisation", partCount)
	}

	// Vérifier qu'il n'y a pas d'enfants
	var childCount int
	db.QueryRow("SELECT COUNT(*) FROM locations WHERE parent_id = ?", locationID).Scan(&childCount)
	if childCount > 0 {
		return fmt.Errorf("impossible de supprimer: %d sous-localisation(s) existent", childCount)
	}

	_, err := db.Exec("DELETE FROM locations WHERE id = ?", locationID)
	return err
}

// GetPartsCount retourne le nombre de pièces dans une localisation (incluant les sous-localisations)
func GetPartsCount(db *sql.DB, locationID int) int {
	var count int

	// Compter les pièces directement dans cette localisation
	db.QueryRow("SELECT COUNT(*) FROM parts WHERE location_id = ?", locationID).Scan(&count)

	// Ajouter les pièces des sous-localisations
	children, _ := ListChildLocations(db, locationID)
	for _, child := range children {
		count += GetPartsCount(db, child.ID)
	}

	return count
}

// PrintLocationTree affiche l'arborescence des localisations
func PrintLocationTree(db *sql.DB) error {
	roots, err := ListRootLocations(db)
	if err != nil {
		return err
	}

	if len(roots) == 0 {
		fmt.Println("Aucune localisation définie.")
		fmt.Println("\nCréez-en une avec:")
		fmt.Println("  recycle loc add \"Atelier Principal\" --type=ZONE")
		return nil
	}

	fmt.Println("\n📍 Arborescence des localisations:")
	fmt.Println(strings.Repeat("─", 50))

	for _, root := range roots {
		printLocationNode(db, root, 0)
	}

	fmt.Println()
	return nil
}

// printLocationNode affiche un nœud de l'arbre et ses enfants récursivement
func printLocationNode(db *sql.DB, loc Location, depth int) {
	indent := strings.Repeat("  ", depth)
	icon := GetLocationIcon(loc.LocType)
	partCount := GetPartsCount(db, loc.ID)

	// Afficher le nœud
	countStr := ""
	if partCount > 0 {
		countStr = fmt.Sprintf(" (%d pièce(s))", partCount)
	}

	if depth == 0 {
		fmt.Printf("%s%s %s [#%d]%s\n", indent, icon, loc.Name, loc.ID, countStr)
	} else {
		fmt.Printf("%s├─ %s %s [#%d]%s\n", indent, icon, loc.Name, loc.ID, countStr)
	}

	// Afficher les enfants
	children, _ := ListChildLocations(db, loc.ID)
	for _, child := range children {
		printLocationNode(db, child, depth+1)
	}
}

// SetPartLocation définit la localisation d'une pièce
func SetPartLocation(db *sql.DB, partID int, locationID int) error {
	// Vérifier que la pièce existe
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM parts WHERE id = ?", partID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("pièce ID %d introuvable", partID)
	}

	// Vérifier que la localisation existe
	_, err = FindLocationByID(db, locationID)
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE parts SET location_id = ? WHERE id = ?", locationID, partID)
	return err
}

// ClearPartLocation supprime la localisation d'une pièce
func ClearPartLocation(db *sql.DB, partID int) error {
	_, err := db.Exec("UPDATE parts SET location_id = NULL WHERE id = ?", partID)
	return err
}

// GetLocationsMap retourne un map des localisations par ID pour affichage batch
func GetLocationsMap(db *sql.DB, locationIDs []int) (map[int]string, error) {
	result := make(map[int]string)

	for _, id := range locationIDs {
		if id == 0 {
			continue
		}
		path, err := GetFullPath(db, id)
		if err == nil {
			result[id] = path
		}
	}

	return result, nil
}
