# Documentation: Partage de Peer Cross-Network

## Vue d'ensemble

Cette fonctionnalité permet le partage de services entre différentes infrastructures NetBird via des expositions cross-network. Elle permet à un peer d'une infrastructure de devenir accessible depuis une autre infrastructure NetBird.

## Architecture

### Composants

1. **CrossNetworkExposeManager** (`management/server/cross_network_expose.go`)
   - Gère le cycle de vie des expositions cross-network
   - Valide et crée de nouvelles expositions
   - Gère la synchronisation avec les clients

2. **CrossNetworkExpose** (Protobuf)
   - Structure de données représentant une exposition
   - Contient les informations source et cible

3. **CrossNetworkPeerConfig** (Protobuf)
   - Configuration envoyée aux clients
   - Permet aux clients de se connecter aux peers cross-network

### Flux de travail

```
┌─────────────────┐
│   Client A      │
│  (Infrastructure A)│
└────────┬────────┘
         │
         │ 1. Request CreateCrossNetworkExpose
         ▼
┌─────────────────┐
│  gRPC Server    │
│ (Management)    │
└────────┬────────┘
         │
         │ 2. Validate & Create
         ▼
┌─────────────────┐
│  Account        │
│  Manager        │
└────────┬────────┘
         │
         │ 3. Update NetworkMap
         ▼
┌─────────────────┐
│   Client B      │
│  (Infrastructure B)│
└─────────────────┘
```

## API gRPC

### CreateCrossNetworkExpose

Crée une nouvelle exposition cross-network.

**Request:**
```protobuf
message CreateCrossNetworkExposeRequest {
  string name = 1;                    // Nom de l'exposition
  string source_peer_id = 2;          // ID du peer source
  string target_account_id = 3;       // ID du compte cible
  string target_network_id = 4;       // ID du réseau cible
  string target_peer_id = 5;          // ID du peer cible
  ExposeProtocol protocol = 6;        // Protocole (HTTP, TCP, UDP, TLS)
  uint32 port = 7;                    // Port source
  uint32 listen_port = 8;             // Port d'écoute (0 = auto)
  repeated string allowed_cidrs = 9;  // CIDRs autorisés
  uint32 ttl_hours = 10;              // TTL en heures (0 = permanent)
}
```

**Response:**
```protobuf
message CreateCrossNetworkExposeResponse {
  CrossNetworkExpose expose = 1;
}
```

### ListCrossNetworkExposes

Liste toutes les expositions cross-network d'un compte.

**Request:**
```protobuf
message Empty {}
```

**Response:**
```protobuf
message ListCrossNetworkExposesResponse {
  repeated CrossNetworkExpose exposes = 1;
}
```

### GetCrossNetworkExpose

Obtient une exposition cross-network par son ID.

**Request:**
```protobuf
message GetCrossNetworkExposeRequest {
  string expose_id = 1;
}
```

**Response:**
```protobuf
message GetCrossNetworkExposeResponse {
  CrossNetworkExpose expose = 1;
}
```

### DeleteCrossNetworkExpose

Supprime une exposition cross-network.

**Request:**
```protobuf
message DeleteCrossNetworkExposeRequest {
  string expose_id = 1;
}
```

**Response:**
```protobuf
message DeleteCrossNetworkExposeResponse {}
```

## Structure de données

### CrossNetworkExpose

```protobuf
message CrossNetworkExpose {
  string id = 1;                      // ID unique
  string name = 2;                    // Nom lisible
  string source_account_id = 3;       // ID du compte source
  string source_network_id = 4;       // ID du réseau source
  string source_peer_id = 5;          // ID du peer source
  string source_peer_name = 6;        // Nom du peer source
  string source_peer_ip = 7;          // IP du peer source
  string target_account_id = 8;       // ID du compte cible
  repeated string target_network_ids = 9;  // IDs des réseaux cibles
  ExposeProtocol protocol = 10;       // Protocole
  uint32 port = 11;                   // Port source
  uint32 listen_port = 12;            // Port d'écoute
  bool port_auto_assigned = 13;       // Auto-assigned?
  bool enabled = 14;                  // Actif?
  uint32 ttl_hours = 15;              // TTL en heures
  uint64 expires_at = 16;             // Timestamp d'expiration
  repeated string allowed_cidrs = 17; // CIDRs autorisés
}
```

### CrossNetworkPeerConfig

```protobuf
message CrossNetworkPeerConfig {
  string id = 1;                      // ID unique
  string name = 2;                    // Nom lisible
  string proxy_endpoint = 3;          // Endpoint proxy
  string target_account_id = 4;       // ID du compte cible
  string target_network_id = 5;       // ID du réseau cible
  string target_peer_id = 6;          // ID du peer cible
  string target_peer_name = 7;        // Nom du peer cible
  string target_peer_ip = 8;          // IP du peer cible
  ExposeProtocol protocol = 9;        // Protocole
  uint32 port = 10;                   // Port
  string mode = 11;                   // Mode (http, tcp, udp, tls)
}
```

## Implémentation

### Fichiers modifiés

1. **management/server/cross_network_expose.go** - Nouveau fichier
   - Définit `CrossNetworkExposeManager`
   - Implémente les méthodes CRUD

2. **management/server/account.go**
   - Ajout du champ `crossNetworkExposeManager`
   - Initialisation dans `BuildManager`
   - Méthode `GetCrossNetworkExposeManager`

3. **management/internals/shared/grpc/server.go**
   - Méthodes gRPC:
     - `CreateCrossNetworkExpose`
     - `ListCrossNetworkExposes`
     - `GetCrossNetworkExpose`
     - `DeleteCrossNetworkExpose`

4. **shared/management/proto/management.proto**
   - Messages protobuf:
     - `CreateCrossNetworkExposeRequest`
     - `CreateCrossNetworkExposeResponse`
     - `GetCrossNetworkExposeRequest`
     - `GetCrossNetworkExposeResponse`
     - `DeleteCrossNetworkExposeRequest`
     - `DeleteCrossNetworkExposeResponse`

### Étapes suivantes

1. **Implémenter le stockage persistant**
   - Ajouter les méthodes au store interface
   - Implémenter les requêtes SQL/NoSQL

2. **Synchronisation client**
   - Mettre à jour `NetworkMap` pour inclure `CrossNetworkPeerConfig`
   - Synchroniser avec les clients connectés

3. **Validation et autorisations**
   - Vérifier que le peer source existe et est connecté
   - Vérifier les permissions utilisateur
   - Valider les CIDRs autorisés

4. **Gestion du cycle de vie**
   - Rendre les expositions éphémères avec expiration
   - Nettoyer les expositions expirées
   - Renouvellement du TTL

5. **Endpoints HTTP REST**
   - API RESTful pour gérer les expositions
   - Authentification et autorisation

## Exemples d'utilisation

### Créer une exposition HTTP

```go
config := &CrossNetworkExposeConfig{
    Name:            "Web Server",
    SourcePeerID:    "peer-123",
    TargetAccountID: "account-456",
    TargetNetworkID: "network-789",
    TargetPeerID:    "",  // Vider pour tout le réseau
    Protocol:        proto.ExposeProtocol_EXPOSE_HTTP,
    Port:            80,
    ListenPort:      0,  // Auto-assigner
    AllowedCIDRs:    []string{"10.0.0.0/8"},
    TTLHours:        0,  // Permanent
}

expose, err := exposeManager.CreateCrossNetworkExpose(ctx, accountID, userID, config)
```

### Lister les expositions

```go
exposes, err := exposeManager.ListCrossNetworkExposes(ctx, accountID)
for _, expose := range exposes {
    fmt.Printf("Expose: %s (%s) -> %s\n", expose.Name, expose.SourcePeerId, expose.TargetAccountId)
}
```

### Supprimer une exposition

```go
err := exposeManager.DeleteCrossNetworkExpose(ctx, accountID, exposeID)
```

## Sécurité

### Validation

- Le peer source doit exister et être connecté
- Les CIDRs doivent être valides
- L'utilisateur doit avoir les permissions appropriées

### Authentification

- Toutes les requêtes doivent être authentifiées
- Le peer key est utilisé pour chiffrer/déchiffrer les messages

### Autorisation

- Les utilisateurs doivent avoir les permissions pour:
  - Créer des expositions
  - Lister les expositions
  - Supprimer des expositions

## Notes

- Cette implémentation est une base pour la fonctionnalité
- Le stockage persistant et la synchronisation complète sont à implémenter
- Les tests unitaires et d'intégration sont nécessaires
