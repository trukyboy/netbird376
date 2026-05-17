#!/bin/bash
set -e

PROVISION_API="https://vp376.net/api/provision"
REPO="https://github.com/trukyboy/netbird376"

echo "╔══════════════════════════════════════╗"
echo "║       Netbird-376 Installer          ║"
echo "╚══════════════════════════════════════╝"

# Vérifier les dépendances
for cmd in curl docker jq git; do
  if ! command -v $cmd &>/dev/null; then
    echo "Installing $cmd..."
    apt-get install -y $cmd 2>/dev/null || yum install -y $cmd 2>/dev/null
  fi
done

# Récupérer l'IP publique du serveur
SERVER_IP=$(curl -s ifconfig.me)
echo "Server IP: $SERVER_IP"

# Demander un sous-domaine à vp376.net
echo "Requesting subdomain from vp376.net..."
RESPONSE=$(curl -s -X POST "$PROVISION_API" \
  -H "Content-Type: application/json" \
  -d "{\"ip\": \"$SERVER_IP\"}")

SUBDOMAIN=$(echo $RESPONSE | jq -r '.subdomain')
DOMAIN=$(echo $RESPONSE | jq -r '.domain')
TOKEN=$(echo $RESPONSE | jq -r '.token')

if [ -z "$DOMAIN" ] || [ "$DOMAIN" = "null" ]; then
  echo "ERROR: Failed to get subdomain from vp376.net"
  exit 1
fi

echo "✅ Subdomain assigned: $DOMAIN"

# Cloner le repo
git clone $REPO /opt/netbird376
cd /opt/netbird376

# Configurer avec le domaine reçu
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" installer/infrastructure_files/docker-compose.yml
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" installer/infrastructure_files/config.yaml
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN/g" installer/infrastructure_files/dashboard.env

# Lancer
cd installer/infrastructure_files
docker compose up -d

echo ""
echo "╔══════════════════════════════════════╗"
echo "║  Installation Complete!              ║"
echo "║  URL: https://$DOMAIN    ║"
echo "╚══════════════════════════════════════╝"
