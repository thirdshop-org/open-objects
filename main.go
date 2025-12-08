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
  dump       Créer une sauvegarde complète (JSON)
  files      Lister les fichiers attachés
  import     Importer des pièces depuis un fichier CSV ou JSON
  list       Lister toutes les pièces
  loc        Gérer les localisations (arborescence atelier)
  restore    Restaurer depuis une sauvegarde JSON
  search     Rechercher des pièces
  templates  Afficher les types de pièces disponibles

Exemples:
  # Gestion des pièces
  recycle add --type=moteur --name="Moteur 12V" --props='{"volts":12, "watts":50}' --loc="Boite Moteurs"
  recycle search --type=roulement --prop="d_int:10..25"
  recycle import --file=stock.csv --type=roulement

  # Gestion des localisations
  recycle loc                                           # Afficher l'arborescence
  recycle loc add "Atelier Vélo" --type=ZONE            # Créer une zone racine
  recycle loc add "Etabli Rouge" --in="Atelier Vélo" --type=FURNITURE
  recycle loc add "Boite Roulements" --in="Etabli Rouge" --type=BOX
  recycle loc move "Boite Roulements" --to="Armoire A"  # Déplacer
  recycle loc set --part=42 --loc="Boite Roulements"    # Localiser une pièce

  # Backup & Restore
  recycle dump                                          # Créer backup_YYYYMMDD_HHMMSS.json
  recycle dump --file=my_backup.json                    # Sauvegarde personnalisée
  recycle restore --file=backup_20231208_143052.json    # Restaurer (avec confirmation)
  recycle restore --file=backup.json --force            # Restaurer sans confirmation

  # Documentation
  recycle attach --id=12 --file=./datasheet.pdf
  recycle files --id=12`)
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
	case "dump":
		if err := cmdDump(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur dump: %v", err)
		}
	case "restore":
		if err := cmdRestore(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur restore: %v", err)
		}
	case "import":
		if err := cmdImport(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur import: %v", err)
		}
	case "list":
		if err := cmdList(db); err != nil {
			log.Fatalf("Erreur list: %v", err)
		}
	case "loc", "location", "locations":
		if err := cmdLoc(db, os.Args[2:]); err != nil {
			log.Fatalf("Erreur loc: %v", err)
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
