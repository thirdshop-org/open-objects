package main

import (
	"database/sql"
	"fmt"
)

// PartRecord représente une ligne de la table parts avec la localisation optionnelle
type PartRecord struct {
	ID         int
	Type       string
	Name       string
	Props      sql.NullString
	LocationID sql.NullInt64
}

// PartMeta pour affichage et QR
type PartMeta struct {
	ID           int
	Type         string
	Name         string
	PropsJSON    string
	LocationID   sql.NullInt64
	LocationPath string
	Found        bool
}

// GetPartMeta retourne les infos d'une pièce par ID
func GetPartMeta(db *sql.DB, id int) (*PartMeta, error) {
	var p PartMeta
	var props sql.NullString
	err := db.QueryRow(`SELECT id, type, name, props, location_id FROM parts WHERE id = ?`, id).
		Scan(&p.ID, &p.Type, &p.Name, &props, &p.LocationID)
	if err == sql.ErrNoRows {
		return &PartMeta{Found: false}, nil
	}
	if err != nil {
		return nil, err
	}
	if props.Valid {
		p.PropsJSON = props.String
	}
	if p.LocationID.Valid {
		path, _ := GetFullPath(db, int(p.LocationID.Int64))
		p.LocationPath = path
	}
	p.Found = true
	return &p, nil
}

// CreatePart insère une pièce et retourne son ID
func CreatePart(db *sql.DB, typeName, name, propsJSON string, locationID *int) (int64, error) {
	var res sql.Result
	var err error
	if locationID != nil {
		res, err = db.Exec("INSERT INTO parts (type, name, props, location_id) VALUES (?, ?, ?, ?)",
			typeName, name, propsJSON, *locationID)
	} else {
		res, err = db.Exec("INSERT INTO parts (type, name, props) VALUES (?, ?, ?)",
			typeName, name, propsJSON)
	}
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

// SearchPartsDB exécute la recherche (CLI + API) en réutilisant la même requête
func SearchPartsDB(db *sql.DB, typeName, nameSearch string, criteria *SearchCriteria) ([]PartRecord, error) {
	var propName, propExact string
	var propMin, propMax float64
	var isRange bool

	if criteria != nil {
		propName = criteria.PropName
		propExact = criteria.ExactVal
		propMin = criteria.MinVal
		propMax = criteria.MaxVal
		isRange = criteria.IsRange
	}

	query := `
		WITH 
		params AS (
			SELECT 
				? AS filter_type,
				? AS filter_name,
				? AS prop_name,
				? AS prop_exact,
				? AS prop_min,
				? AS prop_max,
				? AS is_range
		),
		
		filtered_by_type AS (
			SELECT p.* 
			FROM parts p, params
			WHERE params.filter_type = '' 
			   OR p.type = params.filter_type
		),
		
		filtered_by_name AS (
			SELECT f.* 
			FROM filtered_by_type f, params
			WHERE params.filter_name = '' 
			   OR f.name LIKE '%' || params.filter_name || '%'
		),
		
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
		
		SELECT id, type, name, props, location_id
		FROM filtered_by_prop
		ORDER BY id
	`

	rows, err := db.Query(query, typeName, nameSearch, propName, propExact, propMin, propMax, isRange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []PartRecord
	for rows.Next() {
		var p PartRecord
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Props, &p.LocationID); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}

	return parts, nil
}

// ListAllParts retourne toutes les pièces
func ListAllParts(db *sql.DB) ([]PartRecord, error) {
	rows, err := db.Query("SELECT id, type, name, props, location_id FROM parts ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []PartRecord
	for rows.Next() {
		var p PartRecord
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Props, &p.LocationID); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}

	return parts, nil
}

// MustCriteriaFromProp est un helper pour la CLI/API (retourne nil si prop vide)
func MustCriteriaFromProp(prop string) (*SearchCriteria, error) {
	if prop == "" {
		return nil, nil
	}
	criteria, err := ParseSearchProp(prop)
	if err != nil {
		return nil, fmt.Errorf("prop invalide: %v", err)
	}
	return criteria, nil
}
