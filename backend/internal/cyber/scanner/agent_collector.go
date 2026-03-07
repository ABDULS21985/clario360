package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// AgentPayload is the JSON body posted by endpoint agents to the collector endpoint.
// Agents call POST /api/v1/cyber/assets/agent-checkin with this payload.
type AgentPayload struct {
	AgentID    string         `json:"agent_id"`
	TenantID   string         `json:"tenant_id"`
	Hostname   string         `json:"hostname"`
	IPAddress  string         `json:"ip_address"`
	MACAddress string         `json:"mac_address,omitempty"`
	OS         string         `json:"os"`
	OSVersion  string         `json:"os_version,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

// AgentCollector processes data submitted by endpoint agents via HTTP push.
type AgentCollector struct {
	repo   AssetUpsertRepo
	logger zerolog.Logger
}

// NewAgentCollector creates an AgentCollector.
func NewAgentCollector(repo AssetUpsertRepo, logger zerolog.Logger) *AgentCollector {
	return &AgentCollector{repo: repo, logger: logger}
}

// Type implements Scanner.
func (c *AgentCollector) Type() model.ScanType { return model.ScanTypeAgent }

// Scan is a no-op for the agent collector — agents push data, not the service pulling.
// Agent submissions are handled by ProcessPayload().
func (c *AgentCollector) Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error) {
	return &model.ScanResult{
		Status: model.ScanStatusCompleted,
	}, nil
}

// ProcessPayload processes a single agent submission and upserts the asset into the inventory.
// Returns the asset ID and whether the asset is newly created.
func (c *AgentCollector) ProcessPayload(ctx context.Context, tenantID uuid.UUID, payload *AgentPayload) (uuid.UUID, bool, error) {
	if payload.Hostname == "" && payload.IPAddress == "" {
		return uuid.Nil, false, fmt.Errorf("agent payload requires at least one of: hostname, ip_address")
	}

	// Build optional pointer fields
	var hostname *string
	if payload.Hostname != "" {
		hostname = &payload.Hostname
	}
	var macAddr *string
	if payload.MACAddress != "" {
		macAddr = &payload.MACAddress
	}
	var osVal *string
	if payload.OS != "" {
		osVal = &payload.OS
	}
	var osVer *string
	if payload.OSVersion != "" {
		osVer = &payload.OSVersion
	}

	// Merge agent-specific metadata so it is stored alongside network metadata
	extraMeta := map[string]any{
		"agent_id":        payload.AgentID,
		"agent_checkin":   payload.Timestamp.UTC(),
		"agent_tenant_id": payload.TenantID,
	}
	for k, v := range payload.Metadata {
		extraMeta[k] = v
	}

	d := &model.DiscoveredAsset{
		IPAddress:       payload.IPAddress,
		Hostname:        hostname,
		OS:              osVal,
		OSVersion:       osVer,
		MACAddress:      macAddr,
		AssetType:       model.AssetTypeEndpoint,
		OpenPorts:       []int{},
		Banners:         map[int]string{},
		ExtraMetadata:   extraMeta,
		DiscoverySource: "agent",
	}

	assetID, isNew, err := c.repo.UpsertFromScan(ctx, tenantID, d)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("processing agent payload from %s: %w", payload.AgentID, err)
	}

	c.logger.Debug().
		Str("agent_id", payload.AgentID).
		Str("asset_id", assetID.String()).
		Bool("is_new", isNew).
		Msg("agent payload processed")

	return assetID, isNew, nil
}
