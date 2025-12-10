# Open Objects - Docker Setup

Ce dossier contient les fichiers nécessaires pour containeriser l'application Open Objects avec Docker.

## Fichiers

- `Dockerfile` : Définition de l'image Docker multi-stage pour builder et exécuter l'application
- `docker-compose.yml` : Configuration pour lancer l'application avec Docker Compose
- `.dockerignore` : Fichiers à exclure lors du build
- `build-and-run.sh` : Script automatique pour builder et lancer l'application

## Utilisation

### Avec Docker Compose (recommandé)

```bash
# Construire et lancer l'application
docker-compose up --build

# Ou utiliser le script automatique
./docker/build-and-run.sh

# L'application sera accessible sur http://localhost:8080
```

### Avec Docker directement

```bash
# Construire l'image
docker build -f docker/Dockerfile -t open-objects .

# Lancer le conteneur
docker run -p 8080:8080 open-objects
```

## Volumes

Le `docker-compose.yml` configure un volume nommé `open-objects-data` pour persister la base de données SQLite entre les redémarrages.

## Personnalisation

### Changer le port

Pour changer le port exposé, modifiez le `docker-compose.yml` :

```yaml
ports:
  - "8081:8080"  # Change 8081 par le port désiré
```

### Variables d'environnement

L'application peut être configurée via des variables d'environnement. Ajoutez-les dans le `docker-compose.yml` :

```yaml
environment:
  - PORT=8080
```

## Développement

Pour le développement, vous pouvez monter le code source en volume pour les changements à chaud :

```yaml
volumes:
  - ../:/app
  - /app/open-objects  # Exclure le binaire généré
```