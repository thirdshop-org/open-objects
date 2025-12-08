package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const templatesDir = "templates"
const strictTypes = true // refuser la création de catégories inconnues

// FieldDef définit les métadonnées d'un champ
type FieldDef struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Domain      string `yaml:"domain"`       // dimension, tension, courant, etc.
	DefaultUnit string `yaml:"default_unit"` // mm, V, A, etc.
}

// Template représente un archétype de pièce
type Template struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Fields      map[string]FieldDef `yaml:"fields"`

	// Champs calculés pour rétrocompatibilité
	Required []string `yaml:"-"`
	Optional []string `yaml:"-"`
}

// Templates stocke tous les templates chargés
var Templates = make(map[string]*Template)

// LoadTemplates charge tous les fichiers YAML du dossier templates
func LoadTemplates() error {
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

		// Construire les listes Required et Optional à partir de Fields
		for fieldName, fieldDef := range tmpl.Fields {
			if fieldDef.Required {
				tmpl.Required = append(tmpl.Required, fieldName)
			} else {
				tmpl.Optional = append(tmpl.Optional, fieldName)
			}
		}

		Templates[tmpl.Name] = &tmpl
	}

	return nil
}

// TypeExists indique si un type est connu (présent dans les templates)
func TypeExists(typeName string) bool {
	_, exists := Templates[typeName]
	return exists
}

// ValidateProps vérifie que les propriétés respectent le template et que le type est connu
func ValidateProps(typeName string, props map[string]interface{}) error {
	tmpl, exists := Templates[typeName]
	if !exists {
		if strictTypes {
			return fmt.Errorf("type inconnu: %s (ajoutez un template ou désactivez le mode strict)", typeName)
		}
		return nil // Type libre si le mode strict est désactivé
	}

	for _, req := range tmpl.Required {
		if _, ok := props[req]; !ok {
			return fmt.Errorf("propriété requise manquante: %s (type %s)", req, typeName)
		}
	}

	return nil
}

// GetFieldUnits retourne un map des unités par défaut pour chaque champ d'un template
func GetFieldUnits(typeName string) map[string]string {
	tmpl, exists := Templates[typeName]
	if !exists {
		return nil
	}

	units := make(map[string]string)
	for fieldName, fieldDef := range tmpl.Fields {
		if fieldDef.DefaultUnit != "" {
			units[fieldName] = fieldDef.DefaultUnit
		}
	}

	return units
}

// GetFieldDomain retourne le domaine d'un champ pour un template donné
func GetFieldDomain(typeName, fieldName string) UnitDomain {
	tmpl, exists := Templates[typeName]
	if !exists {
		return DomainNone
	}

	fieldDef, exists := tmpl.Fields[fieldName]
	if !exists {
		return DomainNone
	}

	switch fieldDef.Domain {
	case "dimension":
		return DomainDimension
	case "tension":
		return DomainTension
	case "courant":
		return DomainCourant
	case "resistance":
		return DomainResistance
	case "capacite":
		return DomainCapacite
	case "pression":
		return DomainPression
	case "vitesse_rot":
		return DomainVitesseRot
	case "puissance":
		return DomainPuissance
	default:
		return DomainNone
	}
}
