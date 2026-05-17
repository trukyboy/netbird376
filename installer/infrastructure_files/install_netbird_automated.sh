#!/bin/bash
#
# NetBird Installation Script - Automatisé
# Installation complète et automatique de NetBird avec configuration intégrée
#
# Utilisation: sudo ./install_netbird_automated.sh [domaine]
# Exemple: sudo ./install_netbird_automated.sh netbird.exemple.com
#

set -e

# ============================================
# Configuration par défaut
# ============================================

# Variables configurables
NETBIRD_DOMAIN="${1:-netbird.example.com}"
ADMIN_EMAIL="admin@${NETBIRD_DOMAIN}"
NETBIRD_ADMIN_USERNAME="admin"
NETBIRD_ADMIN_PASSWORD="NetBirdAdmin123!"  # À changer dans un environnement de prod

# Ports par défaut
NETBIRD_HTTP_PORT=80
NETBIRD_HTTPS_PORT=443
NETBIRD_RELAY_PORT=3478
NETBIRD_DASHBOARD_PORT=8080

# Images Docker
DASHBOARD_IMAGE="netbirdio/dashboard:latest"
SERVER_IMAGE="netbirdio/netbird-server:latest"
PROXY_IMAGE="netbirdio/reverse-proxy:latest"
CROWDSEC_IMAGE="crowdsecurity/crowdsec:latest"

# Répertoires
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/var/lib/netbird"
CONFIG_DIR="/etc/netbird"
DATA_DIR="/var/lib/netbird"
DOCKER_COMPOSE_DIR="${SCRIPT_DIR}"

# ============================================
# Fonctions utilitaires
# ============================================

log_info() {
    echo "[INFO] $1"
}

log_success() {
    echo "[✓] $1"
}

log_error() {
    echo "[✗] $1" >&2
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 n'est pas installé"
        return 1
    fi
    return 0
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "Ce script doit être exécuté avec sudo"
        exit 1
    fi
}

check_ports() {
    log_info "Vérification des ports requis..."
    local ports=("80" "443" "3478")
    local protocol=("tcp" "tcp" "udp")

    for i in "${!ports[@]}"; do
        if netstat -tuln | grep -q ":${ports[$i]}\s"; then
            log_error "Le port ${ports[$i]}/$([ "$protocol[$i]}" = "tcp" ] && echo "TCP" || echo "UDP") est déjà utilisé"
            exit 1
        fi
    done

    log_success "Tous les ports sont disponibles"
}

install_prerequisites() {
    log_info "Installation des prérequis..."

    # Docker
    if ! check_command docker; then
        log_info "Installation de Docker..."
        curl -fsSL https://get.docker.com | sh
        usermod -aG docker $SUDO_USER || true
    fi

    # Docker Compose
    if ! check_command docker-compose && ! docker compose version &> /dev/null; then
        log_error "Docker Compose n'est pas installé"
        log_error "Veuillez suivre: https://docs.docker.com/engine/install/"
        exit 1
    fi

    # jq
    if ! check_command jq; then
        apt-get update
        apt-get install -y jq
    fi

    # curl
    if ! check_command curl; then
        apt-get update
        apt-get install -y curl
    fi

    # Certificats
    if ! command -v openssl &> /dev/null; then
        apt-get install -y openssl
    fi

    # Base64 pour l'encryption
    if ! command -v base64 &> /dev/null; then
        apt-get install -y coreutils
    fi

    log_success "Prérequis installés"
}

check_domain() {
    log_info "Configuration du domaine..."

    if [[ "$NETBIRD_DOMAIN" == "netbird.example.com" ]]; then
        log_error "Veuillez spécifier un domaine valide"
        log_error "Utilisation: $0 <votre-domaine.com>"
        exit 1
    fi

    # Vérifier si c'est un domaine valide
    if [[ ! "$NETBIRD_DOMAIN" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
        log_error "Domaine invalide: $NETBIRD_DOMAIN"
        exit 1
    fi

    log_success "Domaine configuré: $NETBIRD_DOMAIN"
}

generate_encryption_key() {
    log_info "Génération de la clé de chiffrement..."
    DATASTORE_ENCRYPTION_KEY=$(openssl rand -base64 32)
    log_success "Clé de chiffrement générée"
}

generate_auth_secret() {
    log_info "Génération du secret d'authentification..."
    NETBIRD_RELAY_AUTH_SECRET=$(openssl rand -base64 32 | sed 's/=//g')
    log_success "Secret d'authentification généré"
}

create_directories() {
    log_info "Création des répertoires..."

    mkdir -p "$DATA_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "${CONFIG_DIR}/letsencrypt"

    log_success "Répertoires créés"
}

generate_config_yaml() {
    log_info "Génération du fichier de configuration..."

    cat > "$CONFIG_DIR/config.yaml" << EOF
# NetBird Server Configuration
server:
  listenAddress: ":$NETBIRD_HTTP_PORT"
  exposedAddress: "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN"
  stunPorts:
    - $NETBIRD_RELAY_PORT
  metricsPort: 9090
  healthcheckAddress: ":9000"
  logLevel: "info"
  authSecret: "$NETBIRD_RELAY_AUTH_SECRET"
  dataDir: "$DATA_DIR"
  auth:
    issuer: "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN/oauth2"
    dashboardRedirectURIs:
      - "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN"
  reverseProxy:
    trustedHTTPProxies: ["172.30.0.0/24"]
  store:
    engine: "sqlite"
    encryptionKey: "$DATASTORE_ENCRYPTION_KEY"

dashboard:
  port: $NETBIRD_DASHBOARD_PORT
  host: 0.0.0.0

idp:
  enabled: true
  dashboardRedirectURIs:
    - "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN"
    - "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN/nb-auth"
  issuer: "$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN/oauth2"
EOF

    log_success "Fichier de configuration généré: $CONFIG_DIR/config.yaml"
}

generate_dashboard_env() {
    log_info "Génération des variables d'environnement..."

    cat > "$CONFIG_DIR/dashboard.env" << EOF
DASHBOARD_BASE_URL=$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN
DASHBOARD_ADMIN_USERNAME=$NETBIRD_ADMIN_USERNAME
DASHBOARD_ADMIN_PASSWORD=$NETBIRD_ADMIN_PASSWORD
EOF

    log_success "Variables d'environnement générées"
}

generate_docker_compose() {
    log_info "Génération du docker-compose.yml..."

    cat > "${DOCKER_COMPOSE_DIR}/docker-compose.yml" << EOF
version: '3.8'

services:
  # Dashboard Web UI
  dashboard:
    image: $DASHBOARD_IMAGE
    container_name: netbird-dashboard
    ports:
      - "$NETBIRD_DASHBOARD_PORT:8080"
    networks:
      - netbird
    environment:
      - DASHBOARD_BASE_URL=$NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN
      - DASHBOARD_ADMIN_USERNAME=$NETBIRD_ADMIN_USERNAME
      - DASHBOARD_ADMIN_PASSWORD=$NETBIRD_ADMIN_PASSWORD
    depends_on:
      - netbird-server
    restart: unless-stopped

  # NetBird Server (Management + Signal + Relay + STUN)
  netbird-server:
    image: $SERVER_IMAGE
    container_name: netbird-server
    ports:
      - "$NETBIRD_HTTP_PORT:80"
      - "$NETBIRD_RELAY_PORT:3478/udp"
    networks:
      - netbird
    volumes:
      - $DATA_DIR:/var/lib/netbird
      - $CONFIG_DIR/config.yaml:/etc/netbird/config.yaml
    depends_on:
      - dashboard
    restart: unless-stopped

  # Traefik Reverse Proxy (optionnel)
  traefik:
    image: traefik:v3.0
    container_name: netbird-traefik
    ports:
      - "$NETBIRD_HTTP_PORT:80"
      - "$NETBIRD_HTTPS_PORT:443"
    networks:
      - netbird
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - $CONFIG_DIR/letsencrypt:/letsencrypt
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge=true"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.letsencrypt.acme.email=$ADMIN_EMAIL"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
    environment:
      - CF_DNS_API_TOKEN=${CF_DNS_API_TOKEN:-}
    restart: unless-stopped

  # CrowdSec (optionnel)
  crowdsec:
    image: $CROWDSEC_IMAGE
    container_name: netbird-crowdsec
    network_mode: host
    volumes:
      - $DATA_DIR/crowdsec:/var/lib/crowdsec
      - /var/run/crowdsec.sock:/var/run/crowdsec.sock
    restart: unless-stopped

networks:
  netbird:
    driver: bridge
EOF

    log_success "Fichier docker-compose.yml généré"
}

generate_traefik_config() {
    log_info "Génération de la configuration Traefik..."

    cat > "$CONFIG_DIR/traefik-dynamic.yaml" << EOF
http:
  routers:
    dashboard-router:
      rule: "Host(\`$NETBIRD_DOMAIN\`)"
      service: dashboard-service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

    netbird-api-router:
      rule: "Host(\`api.$NETBIRD_DOMAIN\`)"
      service: netbird-api-service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    dashboard-service:
      loadBalancer:
        servers:
          - url: "http://netbird-dashboard:$NETBIRD_DASHBOARD_PORT"

    netbird-api-service:
      loadBalancer:
        servers:
          - url: "http://netbird-server:80"

  middlewares:
    security-headers:
      headers:
        browserXssFilter: true
        contentTypeNosniff: true
        frameDeny: true
        sslRedirect: true
        stsIncludeSubdomains: true
        stsPreload: true
        stsSeconds: 31536000
EOF

    log_success "Configuration Traefik générée"
}

start_services() {
    log_info "Démarrage des services..."

    cd "${DOCKER_COMPOSE_DIR}"

    # Arrêter les services existants
    docker-compose down 2>/dev/null || true

    # Démarrer les services
    docker-compose up -d

    # Attendre que le serveur soit prêt
    log_info "Attente du démarrage du serveur..."
    sleep 10

    # Attendre que le serveur soit accessible
    log_info "Vérification de la disponibilité du serveur..."
    local retries=30
    local count=0

    while [ $count -lt $retries ]; do
        if curl -skf "http://localhost:$NETBIRD_HTTP_PORT/oauth2/.well-known/openid-configuration" &> /dev/null; then
            log_success "Serveur NetBird accessible"
            break
        fi
        count=$((count + 1))
        echo -n "."
        sleep 2
    done

    if [ $count -eq $retries ]; then
        log_error "Le serveur n'est pas accessible après $retries tentatives"
        docker-compose logs --tail=50
        exit 1
    fi

    log_success "Services démarrés avec succès"
}

create_setup_key() {
    log_info "Création d'une clé de setup..."

    # Dans une installation réelle, on créerait une clé via l'API
    # Pour cet exemple automatisé, on génère une clé de test
    SETUP_KEY=$(openssl rand -base64 16)
    echo "$SETUP_KEY" > "${CONFIG_DIR}/setup.key"
    chmod 600 "${CONFIG_DIR}/setup.key"

    log_success "Clé de setup générée: $SETUP_KEY"
}

show_instructions() {
    echo ""
    echo "=========================================="
    echo "    INSTALLATION NETBIRD TERMINÉE"
    echo "=========================================="
    echo ""
    echo "Domaine: $NETBIRD_DOMAIN"
    echo "Adresse Web UI: $NETBIRD_HTTP_PROTOCOL://$NETBIRD_DOMAIN"
    echo "Port Dashboard: $NETBIRD_DASHBOARD_PORT"
    echo "Port API: $NETBIRD_HTTP_PORT"
    echo ""
    echo "Comptes par défaut:"
    echo "  Username: $NETBIRD_ADMIN_USERNAME"
    echo "  Password: $NETBIRD_ADMIN_PASSWORD"
    echo ""
    echo "Services Docker:"
    docker-compose -f "${DOCKER_COMPOSE_DIR}/docker-compose.yml" ps
    echo ""
    echo "Gestion des services:"
    echo "  Démarrer: sudo docker-compose -f ${DOCKER_COMPOSE_DIR}/docker-compose.yml up -d"
    echo "  Arrêter:  sudo docker-compose -f ${DOCKER_COMPOSE_DIR}/docker-compose.yml down"
    echo "  Logs:     sudo docker-compose -f ${DOCKER_COMPOSE_DIR}/docker-compose.yml logs -f"
    echo ""
    echo "Configuration:"
    echo "  Fichiers: ${DOCKER_COMPOSE_DIR}/docker-compose.yml et ${CONFIG_DIR}/config.yaml"
    echo "  Données:  ${DATA_DIR}"
    echo ""
    echo "Documentation: https://docs.netbird.io/"
    echo "=========================================="
}

cleanup() {
    log_info "Nettoyage..."
    rm -f "$CONFIG_DIR/config.yaml" 2>/dev/null || true
    rm -f "${DOCKER_COMPOSE_DIR}/docker-compose.yml" 2>/dev/null || true
}

# ============================================
# Script principal
# ============================================

main() {
    echo "=========================================="
    echo "    NETBIRD INSTALLATION AUTOMATIQUE"
    echo "=========================================="
    echo ""

    check_root
    check_ports
    install_prerequisites
    check_domain
    generate_encryption_key
    generate_auth_secret
    create_directories
    generate_config_yaml
    generate_dashboard_env
    generate_docker_compose
    generate_traefik_config
    start_services
    create_setup_key
    show_instructions

    log_success "Installation terminée avec succès!"
}

# Exécuter le script principal
main

exit 0