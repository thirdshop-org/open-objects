package main

import (
	"database/sql"
	"flag"
	"fmt"

	"github.com/google/uuid"
)

func cmdNetwork(db *sql.DB, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: recycle network <invite|add|list>")
	}
	sub := args[0]
	switch sub {
	case "invite":
		token := uuid.New().String()
		fmt.Printf("🔑 Token d'accès (lecture seule): %s\n", token)
		fmt.Println("Configurez vos pairs avec ce token en Authorization: Bearer <token>")
		return nil
	case "add":
		fs := flag.NewFlagSet("network add", flag.ExitOnError)
		name := fs.String("name", "", "Nom du pair")
		url := fs.String("url", "", "URL base (ex: https://stock.voisin.org)")
		token := fs.String("token", "", "Token d'accès (Authorization Bearer)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *name == "" || *url == "" || *token == "" {
			return fmt.Errorf("--name, --url et --token sont requis")
		}
		if err := AddPeer(db, *name, *url, *token); err != nil {
			return err
		}
		fmt.Printf("✓ Pair ajouté: %s (%s)\n", *name, *url)
		return nil
	case "list":
		peers, err := ListPeers(db)
		if err != nil {
			return err
		}
		if len(peers) == 0 {
			fmt.Println("Aucun pair configuré.")
			return nil
		}
		fmt.Println("Pairs configurés:")
		for _, p := range peers {
			fmt.Printf(" - [%d] %s -> %s\n", p.ID, p.Name, p.URL)
		}
		return nil
	default:
		return fmt.Errorf("sous-commande inconnue: %s (invite|add|list)", sub)
	}
}

