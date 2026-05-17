package account

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/netbirdio/netbird/shared/management/proto"
	"github.com/netbirdio/netbird/shared/management/status"
	"github.com/rs/xid"
)

// CrossNetworkExposeManager gère les expositions cross-network entre différentes infrastructures NetBird
type CrossNetworkExposeManager struct {
	accountManager Manager
}

// NewCrossNetworkExposeManager crée un nouveau CrossNetworkExposeManager
func NewCrossNetworkExposeManager(am Manager) *CrossNetworkExposeManager {
	return &CrossNetworkExposeManager{
		accountManager: am,
	}
}

// CrossNetworkExposeConfig représente la configuration pour créer une exposition cross-network
type CrossNetworkExposeConfig struct {
	Name            string
	SourcePeerID    string
	TargetAccountID string
	TargetNetworkID string
	TargetPeerID    string
	Protocol        proto.ExposeProtocol
	Port            uint32
	ListenPort      uint32
	Ports           []uint32
	AllowedCIDRs    []string
	TTLHours        uint32
}

// CreateCrossNetworkExpose crée une nouvelle exposition cross-network
func (m *CrossNetworkExposeManager) CreateCrossNetworkExpose(ctx context.Context, accountID, userID string, config *CrossNetworkExposeConfig) (*proto.CrossNetworkExpose, error) {
	// Validation des paramètres
	if config.Name == "" {
		return nil, status.Errorf(status.InvalidArgument, "name is required")
	}
	if config.SourcePeerID == "" {
		return nil, status.Errorf(status.InvalidArgument, "source_peer_id is required")
	}
	if config.TargetAccountID == "" {
		return nil, status.Errorf(status.InvalidArgument, "target_account_id is required")
	}

	// Générer un ID unique
	exposeID := xid.New().String()

	// Vérifier si le peer source existe dans le compte
	peer, err := m.accountManager.GetPeer(ctx, accountID, config.SourcePeerID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source peer: %w", err)
	}
	if peer == nil {
		return nil, status.Errorf(status.NotFound, "source peer %s not found", config.SourcePeerID)
	}

	// Vérifier si le peer source est connecté
	if !peer.Status.Connected {
		return nil, status.Errorf(status.PreconditionFailed, "source peer %s is not connected", config.SourcePeerID)
	}

	// Vérifier les CIDRs autorisés
	if len(config.AllowedCIDRs) > 0 {
		for _, cidr := range config.AllowedCIDRs {
			if _, _, err := net.ParseCIDR(cidr); err != nil {
				return nil, status.Errorf(status.InvalidArgument, "invalid CIDR %s: %v", cidr, err)
			}
		}
	}

	// Calculer l'heure d'expiration
	var expiresAt uint64
	if config.TTLHours > 0 {
		expiresAt = uint64(time.Now().Add(time.Duration(config.TTLHours) * time.Hour).Unix())
	}

	// Déterminer si le port a été auto-assigné
	portAutoAssigned := false
	listenPort := config.ListenPort
	if listenPort == 0 {
		listenPort = uint32(8000 + rand.Intn(1000)) // Port aléatoire entre 8000 et 8999
		portAutoAssigned = true
	}

	// Créer l'objet CrossNetworkExpose
	expose := &proto.CrossNetworkExpose{
		Id:               exposeID,
		Name:             config.Name,
		SourceAccountId:  accountID,
		SourceNetworkId:  "", // Sera rempli plus tard
		SourcePeerId:     config.SourcePeerID,
		SourcePeerName:   peer.Name,
		SourcePeerIp:     peer.IP.String(),
		TargetAccountId:  config.TargetAccountID,
		TargetNetworkIds: []string{config.TargetNetworkID},
		Protocol:         config.Protocol,
		Port:             config.Port,
		ListenPort:       listenPort,
		PortAutoAssigned: portAutoAssigned,
		Enabled:          true,
		TtlHours:         config.TTLHours,
		ExpiresAt:        expiresAt,
		AllowedCidrs:     config.AllowedCIDRs,
	}

	log.WithContext(ctx).Infof("Created cross-network expose %s for peer %s to account %s", exposeID, config.SourcePeerID, config.TargetAccountID)

	// Save to database
	err = m.accountManager.GetStore().CreateCrossNetworkExpose(ctx, expose)
	if err == nil && len(config.Ports) > 0 {
		_ = m.accountManager.GetStore().UpdateCrossNetworkExposePorts(ctx, exposeID, config.Ports)
	}
	if err != nil {
		return nil, err
	}

	return expose, nil
}

// ListCrossNetworkExposes liste toutes les expositions cross-network d'un compte
func (m *CrossNetworkExposeManager) ListCrossNetworkExposes(ctx context.Context, accountID string) ([]*proto.CrossNetworkExpose, error) {
	exposes, err := m.accountManager.GetStore().GetAccountCrossNetworkExposes(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return exposes, nil
}

// GetCrossNetworkExpose obtient une exposition cross-network par son ID
func (m *CrossNetworkExposeManager) GetCrossNetworkExpose(ctx context.Context, accountID, exposeID string) (*proto.CrossNetworkExpose, error) {
	expose, err := m.accountManager.GetStore().GetCrossNetworkExposeByID(ctx, exposeID)
	if err != nil {
		return nil, err
	}
	return expose, nil
}

// DeleteCrossNetworkExpose supprime une exposition cross-network
func (m *CrossNetworkExposeManager) DeleteCrossNetworkExpose(ctx context.Context, accountID, exposeID string) error {
	err := m.accountManager.GetStore().DeleteCrossNetworkExpose(ctx, exposeID)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Deleted cross-network expose %s", exposeID)
	return nil
}

// UpdateCrossNetworkExpose met à jour une exposition cross-network
func (m *CrossNetworkExposeManager) UpdateCrossNetworkExpose(ctx context.Context, accountID, exposeID string, updates map[string]interface{}) (*proto.CrossNetworkExpose, error) {
	expose, err := m.accountManager.GetStore().GetCrossNetworkExposeByID(ctx, exposeID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		expose.Name = name
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		expose.Enabled = enabled
	}
	if ttlHours, ok := updates["ttl_hours"].(uint32); ok {
		expose.TtlHours = ttlHours
		if ttlHours > 0 {
			expose.ExpiresAt = uint64(time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix())
		}
	}

	err = m.accountManager.GetStore().UpdateCrossNetworkExpose(ctx, expose)
	if err != nil {
		return nil, err
	}

	log.WithContext(ctx).Infof("Updated cross-network expose %s", exposeID)
	return expose, nil
}

// EnableCrossNetworkExpose active une exposition cross-network
func (m *CrossNetworkExposeManager) EnableCrossNetworkExpose(ctx context.Context, accountID, exposeID string) error {
	err := m.accountManager.GetStore().EnableCrossNetworkExpose(ctx, exposeID)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Enabled cross-network expose %s", exposeID)
	return nil
}

// DisableCrossNetworkExpose désactive une exposition cross-network
func (m *CrossNetworkExposeManager) DisableCrossNetworkExpose(ctx context.Context, accountID, exposeID string) error {
	err := m.accountManager.GetStore().DisableCrossNetworkExpose(ctx, exposeID)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Disabled cross-network expose %s", exposeID)
	return nil
}

// renewCrossNetworkExposeTTL renouvelle le TTL d'une exposition cross-network
func (m *CrossNetworkExposeManager) renewCrossNetworkExposeTTL(ctx context.Context, accountID, exposeID string, ttlHours uint32) error {
	err := m.accountManager.GetStore().RenewCrossNetworkExposeTTL(ctx, exposeID, ttlHours)
	if err != nil {
		return err
	}
	log.WithContext(ctx).Infof("Renewed cross-network expose %s TTL to %d hours", exposeID, ttlHours)
	return nil
}

// getCrossNetworkPeerConfig génère la configuration CrossNetworkPeerConfig pour un peer
func (m *CrossNetworkExposeManager) getCrossNetworkPeerConfig(ctx context.Context, accountID, peerID string) []*proto.CrossNetworkPeerConfig {
	configs, err := m.accountManager.GetStore().GetCrossNetworkPeersConfig(ctx, peerID)
	if err != nil {
		log.WithContext(ctx).Errorf("failed to get cross-network peer config: %v", err)
		return []*proto.CrossNetworkPeerConfig{}
	}
	return configs
}

// syncCrossNetworkPeers synchronise les peers cross-network vers le network map
func (m *CrossNetworkExposeManager) syncCrossNetworkPeers(ctx context.Context, accountID string, networkMap *proto.NetworkMap) error {
	err := m.accountManager.GetStore().AddCrossNetworkExposeToNetworkMap(ctx, accountID, networkMap)
	if err != nil {
		log.WithContext(ctx).Errorf("failed to add cross-network exposes to network map: %v", err)
		return err
	}
	return nil
}

// GetCrossNetworkPeersConfig retrieves cross-network peer configs for a peer
func (m *CrossNetworkExposeManager) GetCrossNetworkPeersConfig(ctx context.Context, peerID string) []*proto.CrossNetworkPeerConfig {
	configs, err := m.accountManager.GetStore().GetCrossNetworkPeersConfig(ctx, peerID)
	if err != nil {
		log.WithContext(ctx).Errorf("failed to get cross-network peer config: %v", err)
		return []*proto.CrossNetworkPeerConfig{}
	}
	return configs
}
