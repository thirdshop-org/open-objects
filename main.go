package main

import (
	"fmt"
	"log"
	"os"
)

func printUsage() {
	fmt.Println(`recycle - Gestionnaire de pièces techniques

Usage:
  recycle <commande> [options]

Commandes:
  add        Ajouter une pièce au stock
  attach     Attacher un fichier (PDF, photo) à une pièce
  files      Lister les fichiers attachés
  import     Importer des pièces depuis un fichier CSV ou JSON
  list       Lister toutes les pièces
  search     Rechercher des pièces
  templates  Afficher les types de pièces disponibles

Exemples:
  recycle add --type=moteur --name="Moteur Essuie-Glace" --props='{"volts":12, "watts":50}'
  recycle attach --id=12 --file=./datasheet.pdf
  recycle files --id=12
  recycle import --file=stock.csv --type=roulement
  recycle list
  recycle search --type=roulement --prop="d_int:10..25"
  recycle templates`)
}

func main() {
	// Charger les templates
	if err := LoadTemplates(); err != nil {
		log.Printf("Warning: impossible de charger les templates: %v", err)
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	db, err := InitDB()
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
	case "attach":
		if err := cmdAttach(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur attach: %v", err)
		}
	case "files":
		if err := cmdFiles(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur files: %v", err)
		}
	case "import":
		if err := cmdImport(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur import: %v", err)
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
