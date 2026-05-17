# Cross-Network Expose Implementation

## Summary

This document describes the implementation of cross-network peer sharing functionality for NetBird infrastructure.

## Features Implemented

### 1. Database Model (`management/server/store/cross_network_expose.go`)

Created a new database model `CrossNetworkExpose` with the following fields:
- `ID`: Unique identifier for the exposure
- `Name`: Human-readable name
- `SourceAccountID`: Source account ID
- `SourceNetworkID`: Source network ID
- `SourcePeerID`: Source peer ID
- `SourcePeerName`: Source peer name
- `SourcePeerIP`: Source peer IP address
- `TargetAccountID`: Target account ID
- `TargetNetworkIDs`: Target network IDs (JSON serialized)
- `Protocol`: Exposure protocol
- `Port`: Port number on the source peer
- `ListenPort`: Port number to listen on
- `PortAutoAssigned`: Whether port was auto-assigned
- `Enabled`: Whether the exposure is enabled
- `TTLHours`: TTL in hours
- `ExpiresAt`: Unix timestamp when exposure expires
- `AllowedCIDRs`: Allowed CIDRs (JSON serialized)

### 2. Store Methods

Added the following methods to the `Store` interface and implemented them in `SqlStore`:
- `GetAccountCrossNetworkExposes()`: Lists all cross-network exposes for an account
- `GetCrossNetworkExposeByID()`: Gets a cross-network expose by ID
- `CreateCrossNetworkExpose()`: Creates a new cross-network expose
- `UpdateCrossNetworkExpose()`: Updates an existing cross-network expose
- `DeleteCrossNetworkExpose()`: Deletes a cross-network expose
- `GetExpiredCrossNetworkExposes()`: Gets expired cross-network exposes
- `RenewCrossNetworkExposeTTL()`: Renews the TTL of a cross-network expose
- `EnableCrossNetworkExpose()`: Enables a cross-network expose
- `DisableCrossNetworkExpose()`: Disables a cross-network expose
- `CleanExpiredCrossNetworkExposes()`: Cleans expired cross-network exposes
- `AddCrossNetworkExposeToNetworkMap()`: Adds cross-network exposes to a network map
- `GetCrossNetworkPeersConfig()`: Gets cross-network peer configs for a peer

### 3. Account Manager (`management/server/account/cross_network_expose.go`)

Created the `CrossNetworkExposeManager` with the following methods:
- `CreateCrossNetworkExpose()`: Creates a new cross-network exposure
- `ListCrossNetworkExposes()`: Lists all exposures for an account
- `GetCrossNetworkExpose()`: Gets an exposure by ID
- `DeleteCrossNetworkExpose()`: Deletes an exposure
- `UpdateCrossNetworkExpose()`: Updates an exposure
- `EnableCrossNetworkExpose()`: Enables an exposure
- `DisableCrossNetworkExpose()`: Disables an exposure
- `renewCrossNetworkExposeTTL()`: Renews the TTL of an exposure
- `getCrossNetworkPeerConfig()`: Gets cross-network peer configs
- `syncCrossNetworkPeers()`: Syncs cross-network peers to network map
- `GetCrossNetworkPeersConfig()`: Public method to get cross-network peer configs

### 4. gRPC API (`management/internals/shared/grpc/server.go`)

Added the following gRPC handlers:
- `CreateCrossNetworkExpose()`: Creates a new cross-network exposure
- `ListCrossNetworkExposes()`: Lists all exposures for an account
- `GetCrossNetworkExpose()`: Gets an exposure by ID
- `DeleteCrossNetworkExpose()`: Deletes an exposure

### 5. Protobuf Definitions (`shared/management/proto/management.proto`)

Added the following protobuf messages:
- `CrossNetworkExpose`: Represents a cross-network exposure
- `CrossNetworkPeerConfig`: Represents a cross-network peer config
- `CreateCrossNetworkExposeRequest`: Request to create a cross-network exposure
- `CreateCrossNetworkExposeResponse`: Response after creating a cross-network exposure
- `GetCrossNetworkExposeRequest`: Request to get a cross-network exposure
- `GetCrossNetworkExposeResponse`: Response after getting a cross-network exposure
- `DeleteCrossNetworkExposeRequest`: Request to delete a cross-network exposure
- `DeleteCrossNetworkExposeResponse`: Response after deleting a cross-network exposure
- `ListCrossNetworkExposesResponse`: Response after listing cross-network exposures

### 6. Network Map Integration (`management/internals/shared/grpc/conversion.go`)

Updated the `ToSyncResponse` function to include cross-network peer configs in the network map sent to peers.

### 7. Repository Interface (`management/internals/controllers/network_map/controller/repository.go`)

Added `GetCrossNetworkPeersConfig()` method to the `Repository` interface to support cross-network peer config retrieval.

## Files Modified

1. `management/server/account/manager.go`: Added `GetCrossNetworkExposeManager()` to Manager interface
2. `management/server/account.go`: Added `crossNetworkExposeManager` field and initialization
3. `management/server/account/cross_network_expose.go`: Main implementation file
4. `management/server/store/store.go`: Added cross-network expose methods to Store interface
5. `management/server/store/cross_network_expose.go`: New file with store implementation
6. `management/server/store/sql_store.go`: Added CrossNetworkExpose to AutoMigrate
7. `management/internals/shared/grpc/server.go`: Added gRPC handlers
8. `management/internals/shared/grpc/conversion.go`: Added cross-network peer configs to network map
9. `management/internals/controllers/network_map/controller/repository.go`: Added method to Repository interface
10. `shared/management/proto/management.proto`: Added protobuf messages
11. `management/server/mock_server/account_mock.go`: Regenerated mock file
12. `management/server/store/store_mock.go`: Regenerated mock file

## Compilation Status

- ✅ `./management/...` - Compiles successfully
- ✅ `./shared/...` - Compiles successfully

## Next Steps

1. Implement database migration for the new `cross_network_exposes` table
2. Add tests for the cross-network expose functionality
3. Implement the frontend UI for cross-network expose management (in the dashboard repository)
4. Add cleanup job for expired cross-network exposes
5. Add monitoring and metrics for cross-network exposes
