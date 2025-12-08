package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func cmdAdd(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	typeName := fs.String("type", "", "Type de pièce (ex: roulement, moteur)")
	name := fs.String("name", "", "Nom de la pièce")
	props := fs.String("props", "{}", "Propriétés JSON de la pièce")
	locName := fs.String("loc", "", "Localisation (nom ou ID)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("le nom est requis (--name)")
	}

	// Refuser les types inconnus (taxonomie standard)
	if *typeName != "" && !TypeExists(*typeName) {
		return fmt.Errorf("type '%s' inconnu. Utilisez un template existant (commande 'templates')", *typeName)
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

	// Trouver la localisation si spécifiée
	var locationID *int
	if *locName != "" {
		var loc *Location
		var id int
		if _, err := fmt.Sscanf(*locName, "%d", &id); err == nil {
			loc, _ = FindLocationByID(db, id)
		}
		if loc == nil {
			loc, err = FindLocationByName(db, *locName)
			if err != nil {
				return fmt.Errorf("localisation: %v", err)
			}
		}
		locationID = &loc.ID
	}

	id, err := CreatePart(db, *typeName, *name, string(normalizedJSON), locationID)
	if err != nil {
		return err
	}
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

	// Afficher la localisation
	if locationID != nil {
		path, _ := GetFullPath(db, *locationID)
		fmt.Printf("  📍 Localisation: %s\n", path)
	}

	return nil
}

func cmdList(db *sql.DB) error {
	parts, err := ListAllParts(db)
	if err != nil {
		return err
	}
	return printPartsTableWithAttachments(db, parts, "Total")
}

func cmdSearch(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	typeName := fs.String("type", "", "Filtrer par type de pièce")
	propSearch := fs.String("prop", "", "Recherche par propriété (ex: d_int:10 ou d_int:10..10.5)")
	nameSearch := fs.String("name", "", "Recherche par nom (partiel)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	criteria, err := MustCriteriaFromProp(*propSearch)
	if err != nil {
		return err
	}

	parts, err := SearchPartsDB(db, *typeName, *nameSearch, criteria)
	if err != nil {
		return err
	}

	return printPartsTableWithAttachments(db, parts, "Résultats")
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

func cmdAttach(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("attach", flag.ExitOnError)
	partID := fs.Int("id", 0, "ID de la pièce")
	filePath := fs.String("file", "", "Chemin vers le fichier à attacher")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *partID == 0 {
		return fmt.Errorf("l'ID de la pièce est requis (--id)")
	}
	if *filePath == "" {
		return fmt.Errorf("le fichier est requis (--file)")
	}

	attachment, err := AttachFile(db, *partID, *filePath)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Fichier attaché avec succès\n")
	fmt.Printf("  Pièce ID: %d\n", attachment.PartID)
	fmt.Printf("  Fichier:  %s\n", attachment.Filename)
	fmt.Printf("  Stocké:   %s\n", attachment.Filepath)
	fmt.Printf("  Taille:   %s\n", formatFileSize(attachment.Filesize))

	return nil
}

func cmdFiles(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("files", flag.ExitOnError)
	partID := fs.Int("id", 0, "ID de la pièce (optionnel, liste tous si non spécifié)")
	deleteID := fs.Int("delete", 0, "ID de l'attachement à supprimer")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Suppression d'un attachement
	if *deleteID > 0 {
		if err := DeleteAttachment(db, *deleteID); err != nil {
			return err
		}
		fmt.Printf("✓ Attachement ID %d supprimé\n", *deleteID)
		return nil
	}

	// Lister les fichiers d'une pièce spécifique
	if *partID > 0 {
		return ListPartAttachments(db, *partID)
	}

	// Lister toutes les pièces avec des fichiers attachés
	rows, err := db.Query(`
		SELECT DISTINCT p.id, p.type, p.name, 
			   (SELECT COUNT(*) FROM attachments WHERE part_id = p.id) as attach_count
		FROM parts p
		INNER JOIN attachments a ON a.part_id = p.id
		ORDER BY p.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("\n📎 Pièces avec fichiers attachés:")
	fmt.Println(strings.Repeat("─", 60))

	count := 0
	for rows.Next() {
		var id int
		var typeName, name string
		var attachCount int
		if err := rows.Scan(&id, &typeName, &name, &attachCount); err != nil {
			return err
		}

		fmt.Printf("  [%d] %s - %s (%d fichier(s))\n", id, typeName, name, attachCount)
		count++
	}

	if count == 0 {
		fmt.Println("  Aucune pièce avec fichiers attachés")
	}
	fmt.Println()

	return nil
}

func cmdDump(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	outputFile := fs.String("file", "", "Fichier de sortie (défaut: backup_YYYYMMDD_HHMMSS.json)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Générer un nom de fichier par défaut si non spécifié
	filename := *outputFile
	if filename == "" {
		now := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("backup_%s.json", now)
	}

	// Vérifier que le fichier n'existe pas déjà
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("le fichier %s existe déjà. Utilisez --file pour spécifier un autre nom", filename)
	}

	if err := CreateBackup(db, filename); err != nil {
		return err
	}

	fmt.Printf("\n💾 Sauvegarde disponible: %s\n", filename)
	return nil
}

func cmdRestore(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	inputFile := fs.String("file", "", "Fichier de sauvegarde à restaurer")
	force := fs.Bool("force", false, "Ne pas demander confirmation pour écraser les données")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *inputFile == "" {
		return fmt.Errorf("fichier de sauvegarde requis (--file)")
	}

	// Valider le fichier de backup
	backup, err := ValidateBackupFile(*inputFile)
	if err != nil {
		return fmt.Errorf("fichier de sauvegarde invalide: %v", err)
	}

	fmt.Printf("🔄 Restauration depuis: %s\n", *inputFile)
	fmt.Printf("📊 Sauvegarde: v%s (%s)\n", backup.Version, backup.GeneratedAt[:19])
	fmt.Printf("  📍 Localisations: %d\n", len(backup.Locations))
	fmt.Printf("  🔧 Pièces: %d\n", len(backup.Parts))
	fmt.Printf("  📎 Fichiers: %d\n", len(backup.Attachments))

	// Demander confirmation si pas --force
	if !*force {
		fmt.Print("\n⚠️  ATTENTION: Cela va ÉCRASER toutes les données actuelles!\n")
		fmt.Print("Tapez 'yes' pour continuer: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Restauration annulée.")
			return nil
		}
	}

	if err := RestoreFromBackup(db, *inputFile); err != nil {
		return err
	}

	fmt.Printf("\n✅ Restauration terminée. Redémarrez si nécessaire.\n")
	return nil
}

// cmdLabel génère une étiquette PNG avec QR code sur stdout
func cmdLabel(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("label", flag.ExitOnError)
	id := fs.Int("id", 0, "ID de la pièce")
	url := fs.String("url", "", "URL ou action du QR (défaut: recycle://view/{id})")
	format := fs.String("format", "png", "Format de sortie (png)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id <= 0 {
		return fmt.Errorf("ID requis (--id)")
	}
	if *format != "png" {
		return fmt.Errorf("format '%s' non supporté (seul png est supporté)", *format)
	}

	meta, err := GetPartMeta(db, *id)
	if err != nil {
		return err
	}
	if !meta.Found {
		return fmt.Errorf("pièce ID %d introuvable", *id)
	}

	labelURL := *url
	if labelURL == "" {
		labelURL = DefaultLabelURL(*id)
	}

	if err := GenerateLabelPNG(meta, labelURL, os.Stdout); err != nil {
		return err
	}
	return nil
}

func cmdLoc(db *sql.DB, args []string) error {
	if len(args) == 0 {
		// Sans argument, afficher l'arborescence
		return PrintLocationTree(db)
	}

	subCmd := args[0]

	switch subCmd {
	case "add":
		return cmdLocAdd(db, args[1:])
	case "list", "ls":
		return PrintLocationTree(db)
	case "move", "mv":
		return cmdLocMove(db, args[1:])
	case "delete", "rm":
		return cmdLocDelete(db, args[1:])
	case "set":
		return cmdLocSet(db, args[1:])
	default:
		// Si ce n'est pas une sous-commande, c'est peut-être le nom pour "add"
		return cmdLocAdd(db, args)
	}
}

func cmdLocAdd(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("loc add", flag.ExitOnError)
	parentName := fs.String("in", "", "Nom ou ID de la localisation parente")
	locType := fs.String("type", "BOX", "Type: ZONE, FURNITURE, SHELF, BOX")
	description := fs.String("desc", "", "Description optionnelle")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("nom de la localisation requis\nUsage: recycle loc add \"Nom\" [--in=parent] [--type=TYPE]")
	}

	name := fs.Arg(0)

	// Trouver le parent si spécifié
	var parentID *int
	if *parentName != "" {
		// Essayer d'abord comme ID
		var id int
		if _, err := fmt.Sscanf(*parentName, "%d", &id); err == nil {
			loc, err := FindLocationByID(db, id)
			if err != nil {
				return fmt.Errorf("parent: %v", err)
			}
			parentID = &loc.ID
		} else {
			// Chercher par nom
			loc, err := FindLocationByName(db, *parentName)
			if err != nil {
				return err
			}
			parentID = &loc.ID
		}
	}

	loc, err := CreateLocation(db, name, parentID, *locType, *description)
	if err != nil {
		return err
	}

	icon := GetLocationIcon(loc.LocType)
	fmt.Printf("✓ Localisation créée [ID: %d]\n", loc.ID)
	fmt.Printf("  %s %s (%s)\n", icon, loc.Name, loc.LocType)

	if parentID != nil {
		path, _ := GetFullPath(db, loc.ID)
		fmt.Printf("  📍 Chemin: %s\n", path)
	}

	return nil
}

func cmdLocMove(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("loc move", flag.ExitOnError)
	targetName := fs.String("to", "", "Nouveau parent (nom ou ID, vide = racine)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("ID ou nom de la localisation à déplacer requis\nUsage: recycle loc move <loc> --to=<nouveau_parent>")
	}

	// Trouver la localisation à déplacer
	locArg := fs.Arg(0)
	var loc *Location
	var id int
	if _, err := fmt.Sscanf(locArg, "%d", &id); err == nil {
		loc, _ = FindLocationByID(db, id)
	}
	if loc == nil {
		var err error
		loc, err = FindLocationByName(db, locArg)
		if err != nil {
			return fmt.Errorf("localisation '%s' introuvable", locArg)
		}
	}

	oldPath, _ := GetFullPath(db, loc.ID)

	// Trouver le nouveau parent
	var newParentID *int
	if *targetName != "" {
		var parentID int
		if _, err := fmt.Sscanf(*targetName, "%d", &parentID); err == nil {
			newParentID = &parentID
		} else {
			parent, err := FindLocationByName(db, *targetName)
			if err != nil {
				return fmt.Errorf("nouveau parent: %v", err)
			}
			newParentID = &parent.ID
		}
	}

	if err := MoveLocation(db, loc.ID, newParentID); err != nil {
		return err
	}

	newPath, _ := GetFullPath(db, loc.ID)

	fmt.Printf("✓ Localisation déplacée\n")
	fmt.Printf("  Avant: %s\n", oldPath)
	fmt.Printf("  Après: %s\n", newPath)

	return nil
}

func cmdLocDelete(db *sql.DB, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("ID ou nom de la localisation à supprimer requis")
	}

	locArg := args[0]
	var loc *Location
	var id int
	if _, err := fmt.Sscanf(locArg, "%d", &id); err == nil {
		loc, _ = FindLocationByID(db, id)
	}
	if loc == nil {
		var err error
		loc, err = FindLocationByName(db, locArg)
		if err != nil {
			return fmt.Errorf("localisation '%s' introuvable", locArg)
		}
	}

	path, _ := GetFullPath(db, loc.ID)

	if err := DeleteLocation(db, loc.ID); err != nil {
		return err
	}

	fmt.Printf("✓ Localisation supprimée: %s\n", path)
	return nil
}

func cmdLocSet(db *sql.DB, args []string) error {
	fs := flag.NewFlagSet("loc set", flag.ExitOnError)
	partID := fs.Int("part", 0, "ID de la pièce")
	locName := fs.String("loc", "", "Nom ou ID de la localisation")
	clear := fs.Bool("clear", false, "Supprimer la localisation de la pièce")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *partID == 0 {
		return fmt.Errorf("ID de la pièce requis (--part)")
	}

	if *clear {
		if err := ClearPartLocation(db, *partID); err != nil {
			return err
		}
		fmt.Printf("✓ Localisation supprimée pour la pièce ID %d\n", *partID)
		return nil
	}

	if *locName == "" {
		return fmt.Errorf("localisation requise (--loc) ou utilisez --clear")
	}

	// Trouver la localisation
	var loc *Location
	var id int
	if _, err := fmt.Sscanf(*locName, "%d", &id); err == nil {
		loc, _ = FindLocationByID(db, id)
	}
	if loc == nil {
		var err error
		loc, err = FindLocationByName(db, *locName)
		if err != nil {
			return err
		}
	}

	if err := SetPartLocation(db, *partID, loc.ID); err != nil {
		return err
	}

	path, _ := GetFullPath(db, loc.ID)
	fmt.Printf("✓ Pièce ID %d localisée dans: %s\n", *partID, path)
	return nil
}

// --- Helpers d'affichage ---

func printPartsTableWithAttachments(db *sql.DB, parts []PartRecord, countLabel string) error {
	var partIDs []int
	var locationIDs []int
	for _, p := range parts {
		partIDs = append(partIDs, p.ID)
		if p.LocationID.Valid {
			locationIDs = append(locationIDs, int(p.LocationID.Int64))
		}
	}

	// Récupérer les attachments et localisations
	attachmentsMap, _ := GetAttachmentsForParts(db, partIDs)
	locationsMap, _ := GetLocationsMap(db, locationIDs)

	// Afficher le tableau
	fmt.Println("┌─────┬──────────────┬────────────────────────────┬────────────────────────────────────────┬───────┐")
	fmt.Println("│ ID  │ Type         │ Nom                        │ Propriétés                             │ Docs  │")
	fmt.Println("├─────┼──────────────┼────────────────────────────┼────────────────────────────────────────┼───────┤")

	for _, p := range parts {
		displayType := truncate(p.Type, 12)
		displayName := truncate(p.Name, 26)
		propsStr := "{}"
		if p.Props.Valid {
			propsStr = p.Props.String
		}
		displayProps := truncate(propsStr, 38)

		// Indicateur de fichiers attachés
		docsIndicator := ""
		if attachments, ok := attachmentsMap[p.ID]; ok && len(attachments) > 0 {
			docsIndicator = FormatAttachmentsSummary(attachments)
		}
		docsDisplay := truncate(docsIndicator, 5)

		fmt.Printf("│ %-3d │ %-12s │ %-26s │ %-38s │ %-5s │\n",
			p.ID, displayType, displayName, displayProps, docsDisplay)
	}

	fmt.Println("└─────┴──────────────┴────────────────────────────┴────────────────────────────────────────┴───────┘")
	fmt.Printf("\n%s: %d pièce(s)\n", countLabel, len(parts))

	// Collecter pièces avec docs et pièces avec localisation
	var partsWithDocs []PartRecord
	var partsWithLoc []PartRecord
	for _, p := range parts {
		if attachments, ok := attachmentsMap[p.ID]; ok && len(attachments) > 0 {
			partsWithDocs = append(partsWithDocs, p)
		}
		if p.LocationID.Valid {
			partsWithLoc = append(partsWithLoc, p)
		}
	}

	// Afficher les localisations
	if len(partsWithLoc) > 0 {
		fmt.Println("\n📍 Localisations:")
		for _, p := range partsWithLoc {
			if path, ok := locationsMap[int(p.LocationID.Int64)]; ok {
				fmt.Printf("  [%d] %s: %s\n", p.ID, p.Name, path)
			}
		}
	}

	// Afficher les pièces avec documentation
	if len(partsWithDocs) > 0 {
		fmt.Println("\n📎 Documentation disponible:")
		for _, p := range partsWithDocs {
			attachments := attachmentsMap[p.ID]
			fmt.Printf("  [%d] %s:\n", p.ID, p.Name)
			for _, a := range attachments {
				fmt.Printf("       → %s\n", a.Filepath)
			}
		}
	}

	return nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
