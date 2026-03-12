package llm

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Enabled             bool                      `mapstructure:"enabled"`
	DefaultProvider     string                    `mapstructure:"default_provider"`
	FallbackToRuleBased bool                      `mapstructure:"fallback_to_rule_based"`
	Providers           map[string]ProviderConfig `mapstructure:"providers"`
	Routing             RoutingConfig             `mapstructure:"routing"`
	Safety              SafetyConfig              `mapstructure:"safety"`
	RateLimits          RateLimitDefaults         `mapstructure:"rate_limits"`
	Prompt              PromptConfig              `mapstructure:"prompt"`
	Tokens              TokenConfig               `mapstructure:"tokens"`
}

type ProviderConfig struct {
	APIKeyEnv      string  `mapstructure:"api_key_env"`
	Model          string  `mapstructure:"model"`
	FallbackModel  string  `mapstructure:"fallback_model"`
	BaseURL        string  `mapstructure:"base_url"`
	BaseURLEnv     string  `mapstructure:"base_url_env"`
	DeploymentName string  `mapstructure:"deployment_name"`
	APIVersion     string  `mapstructure:"api_version"`
	Temperature    float64 `mapstructure:"temperature"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
}

type RoutingConfig struct {
	LLMThreshold          float64 `mapstructure:"llm_threshold"`
	MultiIntentDetection  bool    `mapstructure:"multi_intent_detection"`
	ReasoningDetection    bool    `mapstructure:"reasoning_detection"`
}

type SafetyConfig struct {
	MaxToolCallsPerQuery int  `mapstructure:"max_tool_calls_per_query"`
	MaxToolLoopIterations int `mapstructure:"max_tool_loop_iterations"`
	TotalTimeoutSeconds  int  `mapstructure:"total_timeout_seconds"`
	HallucinationGuardEnabled bool `mapstructure:"hallucination_guard_enabled"`
	InjectionGuardEnabled bool `mapstructure:"injection_guard_enabled"`
	PIIFilterEnabled     bool `mapstructure:"pii_filter_enabled"`
}

type RateLimitDefaults struct {
	MaxCallsPerMinute int     `mapstructure:"max_calls_per_minute"`
	MaxCallsPerHour   int     `mapstructure:"max_calls_per_hour"`
	MaxCallsPerDay    int     `mapstructure:"max_calls_per_day"`
	MaxTokensPerDay   int     `mapstructure:"max_tokens_per_day"`
	MaxCostPerDayUSD  float64 `mapstructure:"max_cost_per_day_usd"`
}

type PromptConfig struct {
	ActiveVersion         string `mapstructure:"active_version"`
	IncludeRuleBasedHint  bool   `mapstructure:"include_rule_based_hint"`
}

type TokenConfig struct {
	SystemPromptBudget        int `mapstructure:"system_prompt_budget"`
	ConversationHistoryBudget int `mapstructure:"conversation_history_budget"`
	ToolResultMaxPerCall      int `mapstructure:"tool_result_max_per_call"`
	ResponseMax               int `mapstructure:"response_max"`
}

func LoadConfig() *Config {
	v := viper.New()
	setDefaults(v)

	v.SetConfigType("yaml")
	v.SetConfigName("vciso_llm")
	for _, dir := range []string{
		envOr("VCISO_LLM_CONFIG_DIR", ""),
		filepath.Join("config"),
		filepath.Join("backend", "config"),
	} {
		if strings.TrimSpace(dir) != "" {
			v.AddConfigPath(dir)
		}
	}
	if path := strings.TrimSpace(os.Getenv("VCISO_LLM_CONFIG_PATH")); path != "" {
		v.SetConfigFile(path)
	}
	_ = v.ReadInConfig()

	cfg := &Config{}
	if err := v.UnmarshalKey("vciso.llm", cfg); err != nil || cfg.DefaultProvider == "" {
		_ = v.Unmarshal(cfg)
	}
	applyEnvOverrides(cfg)
	return cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("enabled", true)
	v.SetDefault("default_provider", "openai")
	v.SetDefault("fallback_to_rule_based", true)

	v.SetDefault("providers.openai.api_key_env", "OPENAI_API_KEY")
	v.SetDefault("providers.openai.model", "gpt-4o")
	v.SetDefault("providers.openai.fallback_model", "gpt-4o-mini")
	v.SetDefault("providers.openai.base_url", "https://api.openai.com/v1")
	v.SetDefault("providers.openai.temperature", 0.1)
	v.SetDefault("providers.openai.max_tokens", 4096)
	v.SetDefault("providers.openai.timeout_seconds", 15)

	v.SetDefault("providers.anthropic.api_key_env", "ANTHROPIC_API_KEY")
	v.SetDefault("providers.anthropic.model", "claude-3-5-sonnet-20241022")
	v.SetDefault("providers.anthropic.base_url", "https://api.anthropic.com")
	v.SetDefault("providers.anthropic.temperature", 0.1)
	v.SetDefault("providers.anthropic.max_tokens", 4096)
	v.SetDefault("providers.anthropic.timeout_seconds", 15)

	v.SetDefault("providers.azure.api_key_env", "AZURE_OPENAI_API_KEY")
	v.SetDefault("providers.azure.deployment_name", "gpt-4o")
	v.SetDefault("providers.azure.base_url_env", "AZURE_OPENAI_ENDPOINT")
	v.SetDefault("providers.azure.api_version", "2024-10-01-preview")
	v.SetDefault("providers.azure.temperature", 0.1)
	v.SetDefault("providers.azure.max_tokens", 4096)
	v.SetDefault("providers.azure.timeout_seconds", 15)

	v.SetDefault("providers.local.base_url", "http://localhost:8080/v1")
	v.SetDefault("providers.local.model", "meta-llama/Meta-Llama-3.1-70B-Instruct")
	v.SetDefault("providers.local.temperature", 0.1)
	v.SetDefault("providers.local.max_tokens", 4096)
	v.SetDefault("providers.local.timeout_seconds", 30)

	v.SetDefault("routing.llm_threshold", 0.85)
	v.SetDefault("routing.multi_intent_detection", true)
	v.SetDefault("routing.reasoning_detection", true)

	v.SetDefault("safety.max_tool_calls_per_query", 5)
	v.SetDefault("safety.max_tool_loop_iterations", 5)
	v.SetDefault("safety.total_timeout_seconds", 30)
	v.SetDefault("safety.hallucination_guard_enabled", true)
	v.SetDefault("safety.injection_guard_enabled", true)
	v.SetDefault("safety.pii_filter_enabled", true)

	v.SetDefault("rate_limits.max_calls_per_minute", 10)
	v.SetDefault("rate_limits.max_calls_per_hour", 100)
	v.SetDefault("rate_limits.max_calls_per_day", 500)
	v.SetDefault("rate_limits.max_tokens_per_day", 500000)
	v.SetDefault("rate_limits.max_cost_per_day_usd", 50.0)

	v.SetDefault("prompt.active_version", "v1.0")
	v.SetDefault("prompt.include_rule_based_hint", true)

	v.SetDefault("tokens.system_prompt_budget", 2000)
	v.SetDefault("tokens.conversation_history_budget", 4000)
	v.SetDefault("tokens.tool_result_max_per_call", 10000)
	v.SetDefault("tokens.response_max", 4096)
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if provider := strings.TrimSpace(os.Getenv("VCISO_LLM_PROVIDER")); provider != "" {
		cfg.DefaultProvider = provider
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
