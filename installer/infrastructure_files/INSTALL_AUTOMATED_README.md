# NetBird Installation Automatisée

Script d'installation automatique de NetBird pour déploiement rapide et simplifié.

## 📋 Caractéristiques

- Installation complète et automatisée
- Configuration intégrée avec Traefik reverse proxy
- TLS automatique avec Let's Encrypt
- Services: Dashboard, Server (Management+Signal+Relay), Traefik, CrowdSec (optionnel)
- Aucune interaction requise pendant l'installation

## 🚀 Installation Rapide

### Prérequis

1. **Infrastructure**:
   - Linux (Ubuntu/Debian recommandé)
   - Domaine public configuré avec DNS pointant vers le serveur
   - Ports TCP 80, 443 et UDP 3478 ouverts dans le pare-feu
   - IP publique

2. **Logiciels**:
   - Docker et Docker Compose
   - curl
   - jq

### Utilisation

```bash
# Rendre le script exécutable
sudo chmod +x install_netbird_automated.sh

# Exécuter l'installation
sudo ./install_netbird_automated.sh <votre-domaine.com>

# Exemple
sudo ./install_netbird_automated.sh netbird.exemple.com
```

### Variables Configurables

Avant l'exécution, vous pouvez modifier les variables dans le script:

```bash
NETBIRD_DOMAIN="netbird.exemple.com"          # Votre domaine
NETBIRD_ADMIN_USERNAME="admin"                # Username admin
NETBIRD_ADMIN_PASSWORD="VotreMotDePasse"      # Password admin
NETBIRD_HTTP_PORT=80                          # Port HTTP
NETBIRD_HTTPS_PORT=443                         # Port HTTPS
NETBIRD_RELAY_PORT=3478                       # Port STUN/TURN
```

## 📦 Services Déployés

| Service | Port | Description |
|---------|------|-------------|
| Dashboard | 8080 | Interface Web d'administration |
| NetBird Server | 80/443 | Management + Signal + Relay |
| Traefik | 80/443 | Reverse proxy + TLS |
| CrowdSec | - | Sécurité (optionnel) |

## 🔐 Comptes Par Défaut

Après l'installation:

- **Web UI**: `http://votre-domaine.com`
- **Username**: `admin`
- **Password**: Configurable dans le script (`NETBIRD_ADMIN_PASSWORD`)

## 🛠️ Gestion des Services

```bash
cd /root/netbird/infrastructure_files

# Démarrer les services
sudo docker-compose -f docker-compose.yml up -d

# Arrêter les services
sudo docker-compose -f docker-compose.yml down

# Voir les logs
sudo docker-compose -f docker-compose.yml logs -f

# Redémarrer un service
sudo docker-compose -f docker-compose.yml restart netbird-server

# Voir l'état des services
sudo docker-compose -f docker-compose.yml ps
```

## 📂 Fichiers de Configuration

| Fichier | Description |
|---------|-------------|
| `docker-compose.yml` | Configuration Docker Compose |
| `/etc/netbird/config.yaml` | Configuration du serveur NetBird |
| `/etc/netbird/dashboard.env` | Variables d'environnement Dashboard |
| `/etc/netbird/setup.key` | Clé de setup pour les machines |
| `/etc/netbird/letsencrypt/` | Certificats Let's Encrypt |

## 🔧 Configuration Avancée

### Personnaliser le Port du Serveur

Modifier `NETBIRD_HTTP_PORT` dans le script avant l'exécution.

### Ajouter CrowdSec

Le script inclut CrowdSec par défaut. Pour le désactiver:
1. Supprimer le service `crowdsec` du fichier `docker-compose.yml`
2. Supprimer la section CrowdSec du fichier `traefik-dynamic.yaml`

### Utiliser un Certificat Custom

Au lieu de Let's Encrypt:
1. Placer vos certificats dans `/etc/netbird/letsencrypt/`
2. Modifier la configuration Traefik pour utiliser les certificats existants

### Activer le Proxy

Pour activer le reverse proxy NetBird:
1. Modifier le fichier `docker-compose.yml` pour ajouter le service proxy
2. Générer un token proxy via l'API
3. Configurer les routes dans le Dashboard

## 🐛 Résolution de Problèmes

### Le serveur n'est pas accessible

1. Vérifier que le domaine est correctement configuré en DNS
2. Vérifier que les ports sont ouverts dans le pare-feu
3. Vérifier les logs: `docker-compose logs netbird-server`
4. Vérifier la connectivité: `curl http://localhost:80/oauth2/.well-known/openid-configuration`

### Certificats Let's Encrypt échouent

1. Vérifier que le domaine pointe vers l'IP du serveur
2. Vérifier que le port 80 est accessible depuis l'extérieur
3. Vérifier l'email dans la configuration: `$ADMIN_EMAIL`
4. Consulter les logs Traefik: `docker-compose logs traefik`

### CrowdSec ne fonctionne pas

1. Vérifier que CrowdSec est en cours d'exécution: `docker-compose ps crowdsec`
2. Vérifier les logs: `docker-compose logs crowdsec`
3. Vérifier le port: CrowdSec utilise `host` network mode

## 📝 Commandes Utiles

### Vérifier l'état des services
```bash
docker-compose ps
```

### Voir les logs en temps réel
```bash
docker-compose logs -f netbird-server
docker-compose logs -f traefik
docker-compose logs -f dashboard
```

### Tester l'API
```bash
# Vérifier l'endpoint OIDC
curl -sk https://votre-domaine.com/oauth2/.well-known/openid-configuration

# Vérifier le healthcheck
curl -sk https://votre-domaine.com/health
```

### Backuper les données
```bash
# Backup des configurations
tar -czf netbird-backup-$(date +%Y%m%d).tar.gz \
  docker-compose.yml \
  /etc/netbird/

# Backup des données
tar -czf netbird-data-backup-$(date +%Y%m%d).tar.gz /var/lib/netbird/
```

### Restaurer une installation
```bash
# Restaurer les configurations
tar -xzf netbird-backup-YYYYMMDD.tar.gz -C /root/netbird/infrastructure_files/
docker-compose up -d

# Restaurer les données
tar -xzf netbird-data-backup-YYYYMMDD.tar.gz -C /
docker-compose up -d
```

## 🔒 Sécurité

1. **Changez le mot de passe par défaut** après la première connexion
2. **Activez l'authentification à deux facteurs** dans le Dashboard
3. **Restreignez l'accès** au port 443 via un pare-feu
4. **Utilisez un domaine valide** avec SSL/TLS
5. **Gardez les certificats Let's Encrypt** sécurisés

## 📊 Métriques

Les métriques sont disponibles sur:
- **Endpoint**: `http://localhost:9090/metrics`
- **Dashboard**: `http://votre-domaine.com` (via l'interface Web)

## 📚 Documentation

- **Site officiel**: https://docs.netbird.io/
- **GitHub**: https://github.com/netbirdio/netbird
- **Wiki**: https://github.com/netbirdio/netbird/wiki

## 🔄 Mise à Jour

### Mise à jour des images Docker

```bash
cd /root/netbird/infrastructure_files

# Arrêter les services
docker-compose down

# Mettre à jour les images
docker-compose pull

# Redémarrer les services
docker-compose up -d
```

### Mise à jour du code source

```bash
cd /root/netbird
git pull origin main
make build
```

## 🆘 Support

- **Documentation**: https://docs.netbird.io/
- **GitHub Issues**: https://github.com/netbirdio/netbird/issues
- **Slack**: Rejoindre le canal @netbird
- **Forum**: https://forum.netbird.io/

## 📄 Licence

Ce script est fourni tel quel, sans garantie. NetBird est sous licence BSD-3-Clause (code principal) et AGPLv3 (management/signal/relay).

---

**Dernière mise à jour**: 2026-05-13
**Version du script**: 1.0
**NetBird version**: Main branch