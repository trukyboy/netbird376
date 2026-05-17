package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/netbirdio/netbird/shared/management/proto"
	"github.com/netbirdio/netbird/shared/management/status"
)

// CrossNetworkExpose represents a cross-network exposure persisted in the database
type CrossNetworkExpose struct {
	ID                string `gorm:"primaryKey;size:255"`
	Name              string `gorm:"size:255"`
	SourceAccountID   string `gorm:"size:255;index:idx_cne_source_account"`
	SourceNetworkID   string `gorm:"size:255"`
	SourcePeerID      string `gorm:"size:255"`
	SourcePeerName    string `gorm:"size:255"`
	SourcePeerIP      string `gorm:"size:45"`
	TargetAccountID   string `gorm:"size:255;index:idx_cne_target_account"`
	TargetNetworkIDs  string `gorm:"type:text;serializer:json"` // JSON array of strings
	Protocol          int32  `gorm:"type:int"`
	Port              uint32 `gorm:"type:int"`
	Ports             string `gorm:"type:text;serializer:json"` // JSON array of uint32
	ListenPort        uint32 `gorm:"type:int"`
	PortAutoAssigned  bool   `gorm:"type:boolean"`
	Enabled           bool   `gorm:"type:boolean"`
	TTLHours          uint32 `gorm:"type:int"`
	ExpiresAt         uint64 `gorm:"type:bigint"`
	AllowedCIDRs      string `gorm:"type:text;serializer:json"` // JSON array of strings
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName specifies the table name for CrossNetworkExpose
func (CrossNetworkExpose) TableName() string {
	return "cross_network_exposes"
}

// ToProto converts CrossNetworkExpose to proto.CrossNetworkExpose
func (c *CrossNetworkExpose) ToProto() *proto.CrossNetworkExpose {
	targetNetworkIDs := []string{}
	json.Unmarshal([]byte(c.TargetNetworkIDs), &targetNetworkIDs)

	allowedCIDRs := []string{}
	json.Unmarshal([]byte(c.AllowedCIDRs), &allowedCIDRs)

	return &proto.CrossNetworkExpose{
		Id:               c.ID,
		Name:             c.Name,
		SourceAccountId:  c.SourceAccountID,
		SourceNetworkId:  c.SourceNetworkID,
		SourcePeerId:     c.SourcePeerID,
		SourcePeerName:   c.SourcePeerName,
		SourcePeerIp:     c.SourcePeerIP,
		TargetAccountId:  c.TargetAccountID,
		TargetNetworkIds: targetNetworkIDs,
		Protocol:         proto.ExposeProtocol(c.Protocol),
		Port:             c.Port,
		ListenPort:       c.ListenPort,
		PortAutoAssigned: c.PortAutoAssigned,
		Enabled:          c.Enabled,
		TtlHours:         c.TTLHours,
		ExpiresAt:        c.ExpiresAt,
		AllowedCidrs:     allowedCIDRs,
	}
}

// FromProto converts proto.CrossNetworkExpose to CrossNetworkExpose
func (c *CrossNetworkExpose) FromProto(expose *proto.CrossNetworkExpose) {
	c.ID = expose.Id
	c.Name = expose.Name
	c.SourceAccountID = expose.SourceAccountId
	c.SourceNetworkID = expose.SourceNetworkId
	c.SourcePeerID = expose.SourcePeerId
	c.SourcePeerName = expose.SourcePeerName
	c.SourcePeerIP = expose.SourcePeerIp
	c.TargetAccountID = expose.TargetAccountId

	// Serialize TargetNetworkIDs as JSON
	if expose.TargetNetworkIds != nil {
		data, _ := json.Marshal(expose.TargetNetworkIds)
		c.TargetNetworkIDs = string(data)
	} else {
		c.TargetNetworkIDs = "[]"
	}

	c.Protocol = int32(expose.Protocol)
	c.Port = expose.Port
	c.ListenPort = expose.ListenPort
	c.PortAutoAssigned = expose.PortAutoAssigned
	c.Enabled = expose.Enabled
	c.TTLHours = expose.TtlHours
	c.ExpiresAt = expose.ExpiresAt

	// Serialize AllowedCIDRs as JSON
	if expose.AllowedCidrs != nil {
		data, _ := json.Marshal(expose.AllowedCidrs)
		c.AllowedCIDRs = string(data)
	} else {
		c.AllowedCIDRs = "[]"
	}
}

func decodePorts(s string) []uint32 {
	var ports []uint32
	if s == "" || s == "[]" {
		return ports
	}
	json.Unmarshal([]byte(s), &ports)
	return ports
}

// GetAccountCrossNetworkExposes retrieves all cross-network exposes for an account
func (s *SqlStore) GetAccountCrossNetworkExposes(ctx context.Context, accountID string) ([]*proto.CrossNetworkExpose, error) {
	var exposes []*CrossNetworkExpose
	result := s.db.Where("source_account_id = ? OR target_account_id = ?", accountID, accountID).Find(&exposes)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to get cross-network exposes from store: %v", result.Error)
		return nil, status.Errorf(status.Internal, "failed to get cross-network exposes")
	}

	var resultProto []*proto.CrossNetworkExpose
	for _, e := range exposes {
		resultProto = append(resultProto, e.ToProto())
	}
	return resultProto, nil
}

// GetCrossNetworkExposeByID retrieves a cross-network expose by ID
func (s *SqlStore) GetCrossNetworkExposeByID(ctx context.Context, exposeID string) (*proto.CrossNetworkExpose, error) {
	var expose CrossNetworkExpose
	result := s.db.Where("id = ?", exposeID).First(&expose)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(status.NotFound, "cross-network expose %s not found", exposeID)
		}
		log.WithContext(ctx).Errorf("failed to get cross-network expose from store: %v", result.Error)
		return nil, status.Errorf(status.Internal, "failed to get cross-network expose")
	}
	return expose.ToProto(), nil
}

// CreateCrossNetworkExpose creates a new cross-network expose
func (s *SqlStore) CreateCrossNetworkExpose(ctx context.Context, expose *proto.CrossNetworkExpose) error {
	var cne CrossNetworkExpose
	cne.FromProto(expose)
	result := s.db.Create(&cne)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to create cross-network expose in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to create cross-network expose")
	}
	return nil
}

// UpdateCrossNetworkExpose updates an existing cross-network expose
func (s *SqlStore) UpdateCrossNetworkExpose(ctx context.Context, expose *proto.CrossNetworkExpose) error {
	var cne CrossNetworkExpose
	result := s.db.Where("id = ?", expose.Id).First(&cne)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return status.Errorf(status.NotFound, "cross-network expose %s not found", expose.Id)
		}
		log.WithContext(ctx).Errorf("failed to get cross-network expose from store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to get cross-network expose")
	}

	cne.FromProto(expose)
	result = s.db.Save(&cne)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to update cross-network expose in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to update cross-network expose")
	}
	return nil
}

// DeleteCrossNetworkExpose deletes a cross-network expose
func (s *SqlStore) DeleteCrossNetworkExpose(ctx context.Context, exposeID string) error {
	result := s.db.Where("id = ?", exposeID).Delete(&CrossNetworkExpose{})
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to delete cross-network expose from store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to delete cross-network expose")
	}
	if result.RowsAffected == 0 {
		return status.Errorf(status.NotFound, "cross-network expose %s not found", exposeID)
	}
	return nil
}

// GetExpiredCrossNetworkExposes retrieves cross-network exposes that have expired
func (s *SqlStore) GetExpiredCrossNetworkExposes(ctx context.Context, currentTime uint64) ([]*proto.CrossNetworkExpose, error) {
	var exposes []*CrossNetworkExpose
	result := s.db.Where("expires_at > 0 AND expires_at <= ?", currentTime).Find(&exposes)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to get expired cross-network exposes from store: %v", result.Error)
		return nil, status.Errorf(status.Internal, "failed to get expired cross-network exposes")
	}

	var resultProto []*proto.CrossNetworkExpose
	for _, e := range exposes {
		resultProto = append(resultProto, e.ToProto())
	}
	return resultProto, nil
}

// RenewCrossNetworkExposeTTL renews the TTL of a cross-network expose
func (s *SqlStore) RenewCrossNetworkExposeTTL(ctx context.Context, exposeID string, ttlHours uint32) error {
	var cne CrossNetworkExpose
	result := s.db.Where("id = ?", exposeID).First(&cne)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return status.Errorf(status.NotFound, "cross-network expose %s not found", exposeID)
		}
		log.WithContext(ctx).Errorf("failed to get cross-network expose from store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to get cross-network expose")
	}

	newExpiresAt := uint64(time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix())
	result = s.db.Model(&cne).Update("expires_at", newExpiresAt)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to renew cross-network expose TTL in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to renew cross-network expose TTL")
	}
	return nil
}

// EnableCrossNetworkExpose enables a cross-network expose
func (s *SqlStore) EnableCrossNetworkExpose(ctx context.Context, exposeID string) error {
	result := s.db.Model(&CrossNetworkExpose{}).Where("id = ?", exposeID).Update("enabled", true)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to enable cross-network expose in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to enable cross-network expose")
	}
	if result.RowsAffected == 0 {
		return status.Errorf(status.NotFound, "cross-network expose %s not found", exposeID)
	}
	return nil
}

// DisableCrossNetworkExpose disables a cross-network expose
func (s *SqlStore) DisableCrossNetworkExpose(ctx context.Context, exposeID string) error {
	result := s.db.Model(&CrossNetworkExpose{}).Where("id = ?", exposeID).Update("enabled", false)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to disable cross-network expose in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to disable cross-network expose")
	}
	if result.RowsAffected == 0 {
		return status.Errorf(status.NotFound, "cross-network expose %s not found", exposeID)
	}
	return nil
}

// CleanExpiredCrossNetworkExposes deletes expired cross-network exposes
func (s *SqlStore) CleanExpiredCrossNetworkExposes(ctx context.Context, maxBatch int) (int64, error) {
	var deleted int64
	result := s.db.Where("expires_at > 0 AND expires_at <= ?", uint64(time.Now().Unix())).
		Limit(maxBatch).
		Delete(&CrossNetworkExpose{})
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to clean expired cross-network exposes from store: %v", result.Error)
		return 0, status.Errorf(status.Internal, "failed to clean expired cross-network exposes")
	}
	deleted = result.RowsAffected
	log.WithContext(ctx).Infof("Cleaned %d expired cross-network exposes", deleted)
	return deleted, nil
}

// AddCrossNetworkExposeToNetworkMap adds cross-network exposes to a network map
func (s *SqlStore) AddCrossNetworkExposeToNetworkMap(ctx context.Context, accountID string, networkMap *proto.NetworkMap) error {
	var exposes []*CrossNetworkExpose
	result := s.db.Where("source_account_id = ?", accountID).Where("enabled = ?", true).Find(&exposes)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to get enabled cross-network exposes from store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to get enabled cross-network exposes")
	}

	for _, expose := range exposes {
		// Check if expired
		if expose.ExpiresAt > 0 && uint64(time.Now().Unix()) >= expose.ExpiresAt {
			continue
		}

		// Convert to CrossNetworkPeerConfig format
		targetNetworkIDs := []string{}
		json.Unmarshal([]byte(expose.TargetNetworkIDs), &targetNetworkIDs)

		config := &proto.CrossNetworkPeerConfig{
			Id:               expose.ID,
			Name:             expose.Name,
			ProxyEndpoint:    "proxy.netbird.io:443", // TODO: Get actual proxy endpoint
			TargetAccountId:  expose.TargetAccountID,
			TargetNetworkId:  targetNetworkIDs[0], // Use first network ID
			TargetPeerId:     expose.SourcePeerID,
			TargetPeerName:   expose.SourcePeerName,
			TargetPeerIp:     expose.SourcePeerIP,
			Protocol:         proto.ExposeProtocol(expose.Protocol),
			Port:             expose.Port,
			Mode:             getExposeMode(proto.ExposeProtocol(expose.Protocol)),
		}
		networkMap.CrossNetworkPeers = append(networkMap.CrossNetworkPeers, config)
	}
	return nil
}

// getExposeMode returns the mode string for an expose protocol
func getExposeMode(protocol proto.ExposeProtocol) string {
	switch protocol {
	case proto.ExposeProtocol_EXPOSE_HTTP:
		return "http"
	case proto.ExposeProtocol_EXPOSE_HTTPS:
		return "https"
	case proto.ExposeProtocol_EXPOSE_TCP:
		return "tcp"
	case proto.ExposeProtocol_EXPOSE_UDP:
		return "udp"
	case proto.ExposeProtocol_EXPOSE_TLS:
		return "tls"
	default:
		return "http"
	}
}

// GetCrossNetworkPeersConfig generates cross-network peer configs for a peer
func (s *SqlStore) GetCrossNetworkPeersConfig(ctx context.Context, peerID string) ([]*proto.CrossNetworkPeerConfig, error) {
	var configs []*proto.CrossNetworkPeerConfig

	// Get all enabled cross-network exposes
	var exposes []*CrossNetworkExpose
	result := s.db.Where("enabled = ?", true).Find(&exposes)
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to get cross-network exposes for peer config: %v", result.Error)
		return nil, status.Errorf(status.Internal, "failed to get cross-network exposes")
	}

	for _, expose := range exposes {
		targetNetworkIDs := []string{}
		json.Unmarshal([]byte(expose.TargetNetworkIDs), &targetNetworkIDs)

		config := &proto.CrossNetworkPeerConfig{
			Id:               expose.ID,
			Name:             expose.Name,
			ProxyEndpoint:    "proxy.netbird.io:443", // TODO: Get actual proxy endpoint
			TargetAccountId:  expose.TargetAccountID,
			TargetNetworkId:  targetNetworkIDs[0], // Use first network ID
			TargetPeerId:     expose.SourcePeerID,
			TargetPeerName:   expose.SourcePeerName,
			TargetPeerIp:     expose.SourcePeerIP,
			Protocol:         proto.ExposeProtocol(expose.Protocol),
			Port:             expose.Port,
			Mode:             getExposeMode(proto.ExposeProtocol(expose.Protocol)),
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// UpdateCrossNetworkExposePorts updates the ports of a cross-network expose
func (s *SqlStore) UpdateCrossNetworkExposePorts(ctx context.Context, exposeID string, ports []uint32) error {
	data, _ := json.Marshal(ports)
	result := s.db.Model(&CrossNetworkExpose{}).Where("id = ?", exposeID).Update("ports", string(data))
	if result.Error != nil {
		log.WithContext(ctx).Errorf("failed to update cross-network expose ports in store: %v", result.Error)
		return status.Errorf(status.Internal, "failed to update ports")
	}
	return nil
}
