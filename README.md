# Open Objects

Gestionnaire de pièces techniques avec interface web.

## Installation

### Avec Docker (recommandé)

```bash
docker build -t open-objects .
docker run -p 8080:8080 open-objects
```

L'application sera accessible sur http://localhost:8080

### Depuis les sources

```bash
go mod download
go build -o open-objects .
./open-objects serve
```

## Utilisation

```
recycle - Gestionnaire de pièces techniques

Usage:
  recycle <commande> [options]

Commandes:
  add        Ajouter une pièce au stock
  attach     Attacher un fichier (PDF, photo) à une pièce
  label      Générer une étiquette PNG (QR code) pour une pièce
  serve      Lancer l'API HTTP (mode serveur)
  network    Gérer les pairs fédérés (peers)
  dump       Créer une sauvegarde complète (JSON)
  files      Lister les fichiers attachés
  import     Importer des pièces depuis un fichier CSV ou JSON
  list       Lister toutes les pièces
  loc        Gérer les localisations (arborescence atelier)
  restore    Restaurer depuis une sauvegarde JSON
  search     Rechercher des pièces
  templates  Afficher les types de pièces disponibles
```

## Docker

Le projet inclut un `Dockerfile` multi-stage qui :
- Build l'application Go avec toutes les dépendances
- Utilise une image Alpine finale pour un déploiement léger
- Expose le port 8080 pour l'interface web

## Développement

Le projet utilise :
- Go 1.25.4
- SQLite pour le stockage
- Interface web avec HTMX
- QR codes pour l'identification des pièces