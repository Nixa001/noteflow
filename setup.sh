
// ============================================================================
// Script d'installation et de lancement (save as: setup.sh)
// ============================================================================

#!/bin/bash

echo "🚀 Installation de l'application Bulletins Scolaires"
echo ""

# Vérifier Go
if ! command -v go &> /dev/null; then
    echo "❌ Go n'est pas installé. Veuillez l'installer depuis https://golang.org/"
    exit 1
fi

echo "✅ Go est installé: $(go version)"

# Vérifier wkhtmltopdf
if ! command -v wkhtmltopdf &> /dev/null; then
    echo "⚠️  wkhtmltopdf n'est pas installé"
    echo ""
    echo "Installation selon votre OS:"
    echo "  - Ubuntu/Debian: sudo apt-get install wkhtmltopdf"
    echo "  - MacOS: brew install wkhtmltopdf"
    echo "  - Windows: Télécharger depuis https://wkhtmltopdf.org/downloads.html"
    echo ""
    read -p "Continuer sans wkhtmltopdf ? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✅ wkhtmltopdf est installé"
fi

# Créer la structure de dossiers
echo ""
echo "📁 Création de la structure de dossiers..."
mkdir -p uploads bulletins static/css templates database handlers models middleware services

# Télécharger les dépendances Go
echo ""
echo "📦 Installation des dépendances Go..."
go mod download

echo ""
echo "✅ Installation terminée !"
echo ""
echo "Pour lancer l'application:"
echo "  go run main.go"
echo ""
echo "L'application sera accessible sur: http://localhost:8080"

// ============================================================================
// Makefile pour faciliter le développement
// ============================================================================

.PHONY: run build clean install setup

# Lancer l'application en mode développement
run:
	go run main.go

# Compiler l'application
build:
	go build -o bulletin-app main.go

# Nettoyer les fichiers générés
clean:
	rm -rf bulletins/*.pdf
	rm -rf uploads/*
	rm -f bulletins.db
	rm -f bulletin-app

# Installer les dépendances
install:
	go mod download

# Configuration initiale complète
setup:
	@echo "🚀 Configuration de l'application..."
	@mkdir -p uploads bulletins static/css templates
	@go mod download
	@echo "✅ Configuration terminée !"

# Lancer avec rechargement automatique (nécessite air)
dev:
	@if ! command -v air > /dev/null; then \
		echo "Installation de air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

# Tests
test:
	go test ./...

# Vérification du code
lint:
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installation de golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

// ============================================================================
// .gitignore
// ============================================================================

# Binaires
bulletin-app
*.exe
*.dll
*.so
*.dylib

# Base de données
*.db
*.db-journal

# Fichiers générés
bulletins/*.pdf
uploads/*

# Go
*.test
*.out
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Environnement
.env
.env.local

// ============================================================================
// docker-compose.yml (optionnel, pour déploiement)
// ============================================================================

version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./bulletins:/app/bulletins
      - ./uploads:/app/uploads
      - ./bulletins.db:/app/bulletins.db
    environment:
      - PORT=8080
    restart: unless-stopped

// ============================================================================
// Dockerfile (optionnel, pour déploiement)
// ============================================================================

FROM golang:1.21-alpine AS builder

# Installer wkhtmltopdf
RUN apk add --no-cache wkhtmltopdf

WORKDIR /app

# Copier les fichiers de dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copier le code source
COPY . .

# Compiler l'application
RUN go build -o bulletin-app main.go

FROM alpine:latest

# Installer wkhtmltopdf dans l'image finale
RUN apk add --no-cache wkhtmltopdf ttf-dejavu

WORKDIR /app

# Copier l'exécutable
COPY --from=builder /app/bulletin-app .

# Copier les templates et assets
COPY templates ./templates
COPY static ./static

# Créer les dossiers nécessaires
RUN mkdir -p uploads bulletins

EXPOSE 8080

CMD ["./bulletin-app"]

// ============================================================================
// Guide de déploiement - DEPLOYMENT.md
// ============================================================================

# Guide de Déploiement

## Déploiement Local

### Prérequis
- Go 1.21+
- wkhtmltopdf

### Installation
```bash
git clone <repository>
cd bulletin-scolaire
go mod download
go run main.go
```

Accéder à: http://localhost:8080

## Déploiement avec Docker

### Build
```bash
docker build -t bulletin-scolaire .
```

### Run
```bash
docker run -p 8080:8080 -v $(pwd)/bulletins:/app/bulletins bulletin-scolaire
```

## Déploiement avec Docker Compose

```bash
docker-compose up -d
```

## Déploiement sur VPS (Ubuntu)

### 1. Installer les dépendances
```bash
sudo apt-get update
sudo apt-get install -y golang wkhtmltopdf
```

### 2. Cloner et configurer
```bash
git clone <repository>
cd bulletin-scolaire
go build -o bulletin-app main.go
```

### 3. Créer un service systemd
```bash
sudo nano /etc/systemd/system/bulletin.service
```

Contenu:
```ini
[Unit]
Description=Bulletin Scolaire App
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/bulletin-scolaire
ExecStart=/opt/bulletin-scolaire/bulletin-app
Restart=always

[Install]
WantedBy=multi-user.target
```

### 4. Activer et démarrer
```bash
sudo systemctl enable bulletin.service
sudo systemctl start bulletin.service
```

### 5. Nginx (optionnel)
```nginx
server {
    listen 80;
    server_name votre-domaine.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Sécurité en Production

### 1. HTTPS
Utiliser Let's Encrypt avec certbot:
```bash
sudo certbot --nginx -d votre-domaine.com
```

### 2. Variables d'environnement
Créer un fichier `.env`:
```
DB_PATH=/var/lib/bulletin/bulletins.db
UPLOAD_DIR=/var/lib/bulletin/uploads
BULLETIN_DIR=/var/lib/bulletin/bulletins
SESSION_SECRET=votre-secret-aleatoire
```

### 3. Firewall
```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 4. Backups automatiques
Créer un cron job:
```bash
0 2 * * * /usr/bin/sqlite3 /var/lib/bulletin/bulletins.db ".backup /backups/bulletin-$(date +\%Y\%m\%d).db"
```

## Monitoring

### Logs
```bash
journalctl -u bulletin.service -f
```

### Health Check
Créer un endpoint `/health`:
```go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})
```

## Performance

### Optimisations recommandées
1. Ajouter un cache Redis pour les sessions
2. Utiliser un CDN pour les assets statiques
3. Compresser les réponses (gzip)
4. Limiter le taux de requêtes (rate limiting)

## Maintenance

### Mise à jour
```bash
git pull
go build -o bulletin-app main.go
sudo systemctl restart bulletin.service
```

### Nettoyage
```bash
# Supprimer les anciens bulletins (>30 jours)
find /var/lib/bulletin/bulletins -name "*.pdf" -mtime +30 -delete
```