#!/bin/bash
set -e

MANAGEMENT_URL="${1:-https://vp376.net}"
SETUP_KEY="${2:-}"
BINARY_URL="https://vp376.net/downloads/netbird"

echo "=== Installation NetBird Custom ==="

# 1. Dépendances
echo "[1/5] Installation des dépendances..."
apt-get update -qq
apt-get install -y wireguard-tools systemd-resolved iptables

# 2. Arrêter le service si déjà installé
echo "[2/5] Téléchargement du client NetBird..."
systemctl stop netbird 2>/dev/null || true
netbird service stop 2>/dev/null || true

curl -fsSL "$BINARY_URL" -o /usr/local/bin/netbird
chmod +x /usr/local/bin/netbird

# 3. Installer le service
echo "[3/5] Installation du service..."
netbird service install 2>/dev/null || true

# 4. Démarrer le service
echo "[4/5] Démarrage du service..."
netbird service start

# 5. Connecter
if [ -n "$SETUP_KEY" ]; then
    echo "[5/5] Connexion au serveur $MANAGEMENT_URL..."
    sleep 2
    netbird up --management-url "$MANAGEMENT_URL" --setup-key "$SETUP_KEY"
else
    echo "[5/5] Pour connecter la machine:"
    echo "  netbird up --management-url $MANAGEMENT_URL --setup-key <SETUP_KEY>"
fi

echo "=== Installation terminée ! ==="
