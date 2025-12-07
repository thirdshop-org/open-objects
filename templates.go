package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const templatesDir = "templates"

// Template représente un archétype de pièce
type Template struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Required    []string `yaml:"required"`
	Optional    []string `yaml:"optional"`
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

		Templates[tmpl.Name] = &tmpl
	}

	return nil
}

// ValidateProps vérifie que les propriétés respectent le template
func ValidateProps(typeName string, props map[string]interface{}) error {
	tmpl, exists := Templates[typeName]
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

