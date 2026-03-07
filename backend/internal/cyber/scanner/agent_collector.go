package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// AgentPayload is the JSON body posted by endpoint agents to the collector endpoint.
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

// AgentCollector processes data submitted by endpoint agents.
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

// Scan is a no-op for the agent collector — agents push data via HTTP, not pull.
// Agent submissions are handled by ProcessPayload().
func (c *AgentCollector) Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error) {
	return &model.ScanResult{
		Status: model.ScanStatusCompleted,
		Errors: []string{"agent collector receives push data — call ProcessPayload instead"},
	}, nil
}

// ProcessPayload processes a single agent submission and upserts the asset.
func (c *AgentCollector) ProcessPayload(ctx context.Context, tenantID uuid.UUID, payload *AgentPayload) (uuid.UUID, bool, error) {
	if payload.Hostname == "" && payload.IPAddress == "" {
		return uuid.Nil, false, fmt.Errorf("agent payload missing both hostname and IP address")
	}

	meta, _ := json.Marshal(map[string]any{
		"agent_id":   payload.AgentID,
		"agent_data": payload.Metadata,
	})

	hostname := &payload.Hostname
	if payload.Hostname == "" {
		hostname = nil
	}
	macAddr := &payload.MACAddress
	if payload.MACAddress == "" {
		macAddr = nil
	}
	osVal := &payload.OS
	if payload.OS == "" {
		osVal = nil
	}
	osVer := &payload.OSVersion
	if payload.OSVersion == "" {
		osVer = nil
	}

	d := &model.DiscoveredAsset{
		IPAddress:  payload.IPAddress,
		Hostname:   hostname,
		OS:         osVal,
		OSVersion:  osVer,
		AssetType:  model.AssetTypeEndpoint,
		Banners:    map[int]string{},
	}
	_ = macAddr // stored in metadata for now
	_ = meta

	return c.repo.UpsertFromScan(ctx, tenantID, d)
}
