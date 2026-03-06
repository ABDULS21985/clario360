package ratelimit

import (
	"time"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

// GroupLimit defines rate limit settings for an endpoint group.
type GroupLimit struct {
	RequestsPerWindow int
	Window            time.Duration
	BurstPerSecond    int
}

// Config holds the full rate limit configuration.
type Config struct {
	Groups map[gwconfig.EndpointGroup]GroupLimit
}

// DefaultConfig returns the default rate limit configuration per endpoint group.
func DefaultConfig() Config {
	return Config{
		Groups: map[gwconfig.EndpointGroup]GroupLimit{
			gwconfig.EndpointGroupAuth: {
				RequestsPerWindow: 20,
				Window:            1 * time.Minute,
				BurstPerSecond:    5,
			},
			gwconfig.EndpointGroupRead: {
				RequestsPerWindow: 2000,
				Window:            1 * time.Minute,
				BurstPerSecond:    100,
			},
			gwconfig.EndpointGroupWrite: {
				RequestsPerWindow: 500,
				Window:            1 * time.Minute,
				BurstPerSecond:    50,
			},
			gwconfig.EndpointGroupAdmin: {
				RequestsPerWindow: 100,
				Window:            1 * time.Minute,
				BurstPerSecond:    20,
			},
			gwconfig.EndpointGroupUpload: {
				RequestsPerWindow: 50,
				Window:            1 * time.Minute,
				BurstPerSecond:    10,
			},
			gwconfig.EndpointGroupWS: {
				RequestsPerWindow: 10,
				Window:            1 * time.Minute,
				BurstPerSecond:    2,
			},
		},
	}
}

// ConfigFromGateway builds a Config from GatewayConfig rate limit settings.
func ConfigFromGateway(
	authPerMin, readPerMin, writePerMin, adminPerMin, uploadPerMin, wsPerMin int,
) Config {
	return Config{
		Groups: map[gwconfig.EndpointGroup]GroupLimit{
			gwconfig.EndpointGroupAuth:   {RequestsPerWindow: authPerMin, Window: time.Minute, BurstPerSecond: authPerMin / 5},
			gwconfig.EndpointGroupRead:   {RequestsPerWindow: readPerMin, Window: time.Minute, BurstPerSecond: readPerMin / 10},
			gwconfig.EndpointGroupWrite:  {RequestsPerWindow: writePerMin, Window: time.Minute, BurstPerSecond: writePerMin / 5},
			gwconfig.EndpointGroupAdmin:  {RequestsPerWindow: adminPerMin, Window: time.Minute, BurstPerSecond: adminPerMin / 5},
			gwconfig.EndpointGroupUpload: {RequestsPerWindow: uploadPerMin, Window: time.Minute, BurstPerSecond: uploadPerMin / 5},
			gwconfig.EndpointGroupWS:     {RequestsPerWindow: wsPerMin, Window: time.Minute, BurstPerSecond: 2},
		},
	}
}

// GetLimit returns the rate limit for a given endpoint group, falling back to write defaults.
func (c Config) GetLimit(group gwconfig.EndpointGroup) GroupLimit {
	if limit, ok := c.Groups[group]; ok {
		return limit
	}
	return c.Groups[gwconfig.EndpointGroupWrite]
}
