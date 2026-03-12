package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
	llmdto "github.com/clario360/platform/internal/cyber/vciso/llm/dto"
)

type Manager struct {
	cfg       *llmcfg.Config
	overrides sync.Map
}

type TenantOverride struct {
	Provider    string
	Model       string
	Temperature float64
}

func NewManager(cfg *llmcfg.Config) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) Resolve(ctx context.Context, tenantID uuid.UUID) (LLMProvider, error) {
	_ = ctx
	if m == nil || m.cfg == nil {
		return nil, fmt.Errorf("llm provider manager is unavailable")
	}
	override := m.getOverride(tenantID)
	providerName := m.cfg.DefaultProvider
	if override.Provider != "" {
		providerName = override.Provider
	}
	providerCfg, ok := m.cfg.Providers[providerName]
	if !ok {
		return nil, fmt.Errorf("llm provider %q is not configured", providerName)
	}
	if override.Model != "" {
		providerCfg.Model = override.Model
		providerCfg.DeploymentName = override.Model
	}
	if override.Temperature > 0 {
		providerCfg.Temperature = override.Temperature
	}
	switch providerName {
	case "openai":
		return NewOpenAIProvider(providerCfg), nil
	case "anthropic":
		return NewAnthropicProvider(providerCfg), nil
	case "azure":
		return NewAzureProvider(providerCfg), nil
	case "local":
		return NewLocalProvider(providerCfg), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", providerName)
	}
}

func (m *Manager) GetConfig(tenantID uuid.UUID) TenantOverride {
	override := m.getOverride(tenantID)
	if override.Provider == "" && m != nil && m.cfg != nil {
		override.Provider = m.cfg.DefaultProvider
		if cfg, ok := m.cfg.Providers[override.Provider]; ok {
			override.Model = cfg.Model
			if override.Temperature == 0 {
				override.Temperature = cfg.Temperature
			}
		}
	}
	return override
}

func (m *Manager) UpdateConfig(tenantID uuid.UUID, req llmdto.UpdateConfigRequest) TenantOverride {
	override := TenantOverride{
		Provider:    req.Provider,
		Model:       req.Model,
		Temperature: req.Temperature,
	}
	m.overrides.Store(tenantID.String(), override)
	return m.GetConfig(tenantID)
}

func (m *Manager) getOverride(tenantID uuid.UUID) TenantOverride {
	if m == nil {
		return TenantOverride{}
	}
	value, _ := m.overrides.Load(tenantID.String())
	override, _ := value.(TenantOverride)
	return override
}
