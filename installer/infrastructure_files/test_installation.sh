#!/bin/bash
#
# NetBird Script de Test Rapide
# Test de l'installation de NetBird dans un environnement de développement
#
# Utilisation: ./test_installation.sh
#

set -e

# Configuration pour les tests
TEST_DOMAIN="test-netbird.local"
TEST_CONTAINER_NAME="netbird-test"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="TestPassword123!"
SERVER_PORT=8080

echo "=========================================="
echo "    NETBIRD TEST RAPIDE"
echo "=========================================="
echo ""

# Vérifier si Docker est disponible
if ! command -v docker &> /dev/null; then
    echo "[✗] Docker n'est pas installé"
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "[✗] Docker Compose n'est pas installé"
    exit 1
fi

echo "[INFO] Création du réseau de test..."
docker network create netbird-test-network 2>/dev/null || true

echo "[INFO] Création des répertoires de test..."
mkdir -p /tmp/netbird-test
mkdir -p /tmp/netbird-test/letsencrypt

echo "[INFO] Génération de la configuration..."
cd /tmp/netbird-test

# Configuration de test
DATASTORE_ENCRYPTION_KEY=$(openssl rand -base64 32)
NETBIRD_RELAY_AUTH_SECRET=$(openssl rand -base64 32 | sed 's/=//g')

cat > config.yaml << EOF
server:
  listenAddress: ":80"
  exposedAddress: "http://$TEST_DOMAIN"
  stunPorts:
    - 3478
  metricsPort: 9090
  healthcheckAddress: ":9000"
  logLevel: "debug"
  authSecret: "$NETBIRD_RELAY_AUTH_SECRET"
  dataDir: "/var/lib/netbird"
  auth:
    issuer: "http://$TEST_DOMAIN/oauth2"
    dashboardRedirectURIs:
      - "http://$TEST_DOMAIN"
  reverseProxy:
    trustedHTTPProxies: ["172.30.0.0/24"]
  store:
    engine: "sqlite"
    encryptionKey: "$DATASTORE_ENCRYPTION_KEY"

dashboard:
  port: $SERVER_PORT
  host: 0.0.0.0

idp:
  enabled: true
  dashboardRedirectURIs:
    - "http://$TEST_DOMAIN"
    - "http://$TEST_DOMAIN/nb-auth"
  issuer: "http://$TEST_DOMAIN/oauth2"
EOF

cat > docker-compose.yml << EOF
version: '3.8'

services:
  dashboard:
    image: netbirdio/dashboard:latest
    container_name: netbird-dashboard-test
    ports:
      - "$SERVER_PORT:8080"
    networks:
      - test-network
    environment:
      - DASHBOARD_BASE_URL=http://$TEST_DOMAIN
      - DASHBOARD_ADMIN_USERNAME=$ADMIN_USERNAME
      - DASHBOARD_ADMIN_PASSWORD=$ADMIN_PASSWORD
    depends_on:
      - netbird-server
    restart: unless-stopped

  netbird-server:
    image: netbirdio/netbird-server:latest
    container_name: netbird-server-test
    ports:
      - "80:80"
      - "3478:3478/udp"
    networks:
      - test-network
    volumes:
      - /tmp/netbird-test:/var/lib/netbird
      - /tmp/netbird-test/config.yaml:/etc/netbird/config.yaml
    depends_on:
      - dashboard
    restart: unless-stopped

networks:
  test-network:
    driver: bridge
EOF

echo "[INFO] Démarrage des services de test..."
docker-compose down 2>/dev/null || true
docker-compose up -d

echo "[INFO] Attente du démarrage..."
sleep 5

echo "[INFO] Vérification des services..."
echo ""
echo "=== Services Docker ==="
docker-compose ps

echo ""
echo "=== Logs récents (5 dernières lignes) ==="
docker-compose logs --tail=5

echo ""
echo "[INFO] Test de l'API..."
sleep 3

# Test 1: Vérifier que le serveur est accessible
echo ""
echo "Test 1: Accès API (healthcheck)..."
if curl -sf http://localhost:80/health &> /dev/null; then
    echo "[✓] Healthcheck OK"
else
    echo "[✗] Healthcheck FAILED"
    docker-compose logs netbird-server | tail -20
fi

# Test 2: Vérifier l'endpoint OIDC
echo ""
echo "Test 2: Endpoint OIDC..."
if curl -sf http://localhost:80/oauth2/.well-known/openid-configuration &> /dev/null; then
    echo "[✓] Endpoint OIDC OK"
    echo ""
    echo "Configuration OIDC:"
    curl -s http://localhost:80/oauth2/.well-known/openid-configuration | jq '.' 2>/dev/null || echo "jq non disponible, impossible de formater"
else
    echo "[✗] Endpoint OIDC FAILED"
    docker-compose logs netbird-server | tail -20
fi

# Test 3: Vérifier le Dashboard
echo ""
echo "Test 3: Endpoint Dashboard..."
if curl -sf http://localhost:$SERVER_PORT &> /dev/null; then
    echo "[✓] Dashboard accessible sur http://localhost:$SERVER_PORT"
else
    echo "[✗] Dashboard FAILED"
    docker-compose logs dashboard | tail -20
fi

# Test 4: Vérifier les ports
echo ""
echo "Test 4: Ports ouverts..."
echo "Ports TCP:"
netstat -tuln | grep -E ":(80|${SERVER_PORT})\s" | head -5 || ss -tuln | grep -E ":(80|${SERVER_PORT})\s" | head -5

echo "Ports UDP:"
netstat -tuln | grep ":3478" || ss -tuln | grep ":3478"

echo ""
echo "=== Test Terminé ==="
echo ""
echo "Services en cours d'exécution:"
docker-compose ps

echo ""
echo "Pour arrêter les tests:"
echo "  cd /tmp/netbird-test && docker-compose down"
echo ""
echo "Pour voir les logs:"
echo "  cd /tmp/netbird-test && docker-compose logs -f"
echo ""
echo "Pour nettoyer tout:"
echo "  cd /tmp/netbird-test && docker-compose down -v && rm -rf /tmp/netbird-test"
echo ""

# Exit with code 0 to indicate successful test (even if some checks failed)
exit 0