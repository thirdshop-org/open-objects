# Open Objects Core (Protocol & CLI)

> **"Pour réparer demain, il faut savoir ce que l'on possède aujourd'hui."**

Open Objects est un moteur de base de données **open-source**, **local-first** et **résilient** conçu pour indexer des pièces détachées techniques (moteurs, roulements, fixations) non pas par leur marque, mais par leurs **propriétés physiques**.

L'objectif est de permettre la réparation en mode dégradé (pénurie, coupure réseau) en identifiant des pièces compatibles dans un stock de récupération hétéroclite.

## 🎯 La Mission

Dans un monde où la logistique mondiale est fragile, savoir qu'un "Lave-linge Samsung Model X" contient un "Moteur compatible avec une pompe à eau de puits" devient une information vitale.

Open Objects n'est pas un simple inventaire. C'est un **moteur de recherche de compatibilité**.
*   Ce n'est pas : "Avez-vous la pièce réf #1234 ?"
*   C'est : "J'ai besoin d'un moteur, 12V/24V, axe de 5mm (±0.1mm). Qu'avons-nous en stock qui correspond ?"

## 🛠 Stack Technique

Choix technologiques dictés par la **sobriété**, la **portabilité** et la **pérénité**.

*   **Langage :** [Go (Golang)](https://go.dev/). Permet de compiler un **binaire statique unique** (pas de dépendances à installer). Fonctionne sur Linux, Windows, macOS, Raspberry Pi, et même Android (via Termux).
*   **Base de Données :** [SQLite](https://www.sqlite.org/). Le standard mondial du stockage local. Un seul fichier `.db` facile à sauvegarder, dupliquer ou copier sur une clé USB.
*   **Interface :** CLI (Command Line Interface) en priorité pour la robustesse et l'automatisation. Une interface Web locale (localhost) sera ajoutée par la suite.

## 🚀 Fonctionnalités Clés (Vision)

1.  **Architecture "Schema-less" Hybride :** Utilisation de JSONB dans SQLite pour s'adapter à n'importe quel type d'objet (un roulement a un diamètre, une batterie a un voltage).
2.  **Matching Flou (Fuzzy Logic) :** Algorithmes de recherche capable de gérer des tolérances.
3.  **Local-First & Offline :** Aucune connexion internet requise. Tout tient sur une clé USB.
4.  **Protocole d'Échange :** Import/Export simple (JSON/CSV) pour partager des bases de connaissances entre communautés (Emmaüs ↔ FabLab).

## 🗺️ Feuille de Route & Validation (Roadmap)

Le développement est découpé en phases strictes pour valider l'utilité à chaque étape.

### Phase 1 : Le MVP "Moteur" (Focus actuel)
Objectif : Prouver que l'on peut stocker et retrouver une pièce technique via le terminal.

- [ ] **Initialisation** : Structure du projet Go + Driver SQLite (`mattn/go-sqlite3` ou moderne `modernc.org/sqlite`).
- [ ] **Modèle de Données** : Création de la table `parts` avec support JSON pour les propriétés dynamiques.
- [ ] **Commande `add`** : Implémenter l'ajout d'une pièce avec attributs libres.
    - *Test :* `Open Objects add --name="Moteur Essuie-Glace" --props='{"volts":12, "axe":6}'`
- [ ] **Commande `list`** : Lister tout le stock brut.

### Phase 2 : La Recherche Intelligente
Objectif : Rendre l'outil utile pour un technicien.

- [ ] **Système de Templates (Archétypes)** : Définir des fichiers YAML pour contraindre les types (ex: un *roulement* demande obligatoirement *d_int*, *d_ext*).
- [ ] **Commande `search` (Exacte)** : Retrouver via SQL simple.
- [ ] **Commande `search` (Range)** : Le cœur du projet.
    - *Test :* `Open Objects search --type=roulement --prop="d_int:10..10.5"` (Doit trouver un roulement de 10.2mm).

### Phase 3 : Confort & Accessibilité
Objectif : Rendre l'outil utilisable par des bénévoles non-devs.

- [ ] **Serveur Web Embarqué** : Le binaire lance un serveur HTTP local sur le port 8080.
- [ ] **Web UI (v0.1)** : Formulaires HTML simples pour `add` et `search` sans passer par le terminal.
- [ ] **Documentation Utilisateur** : Guide PDF imprimable pour expliquer comment mesurer une pièce (pied à coulisse, multimètre).

## 💻 Installation (Développement)

```bash
# Cloner le repo
git clone https://github.com/votre-username/Open Objects-core.git

# Aller dans le dossier
cd Open Objects-core

# Lancer sans compiler
go run main.go

# Compiler le binaire
go build -o Open Objects
```

## 🤝 Contribuer

Ce projet vise à devenir un bien commun numérique.
*   **Code :** Go (Architecture Hexagonale ou Clean Architecture recommandée).
*   **Données :** Nous cherchons des experts métiers (vélo, électroménager) pour définir les archétypes de pièces.

---

**Licence :** AGPL-3.0 (Garantit que le code reste ouvert et libre pour toujours).