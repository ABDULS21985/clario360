package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

// SubscriptionTier represents a tenant's subscription level.
type SubscriptionTier string

const (
	TierFree         SubscriptionTier = "free"
	TierProfessional SubscriptionTier = "professional"
	TierEnterprise   SubscriptionTier = "enterprise"
)

// TenantRateLimits defines per-operation rate limits for a subscription tier.
// All RPM (requests per minute) values define the sustained rate.
// BurstMultiplier is applied to calculate short-burst allowances.
type TenantRateLimits struct {
	DefaultRPM      int
	ReadRPM         int
	WriteRPM        int
	BulkRPM         int
	ExportRPM       int
	BurstMultiplier float64
}

// TierLimits defines default rate limits per subscription tier.
// These represent the platform-wide defaults. Specific tenants may have custom
// overrides stored in Redis under the key tenant_tier:<tenant_id>.
var TierLimits = map[SubscriptionTier]TenantRateLimits{
	TierFree: {
		DefaultRPM:      100,
		ReadRPM:         200,
		WriteRPM:        50,
		BulkRPM:         5,
		ExportRPM:       10,
		BurstMultiplier: 1.2,
	},
	TierProfessional: {
		DefaultRPM:      1000,
		ReadRPM:         2000,
		WriteRPM:        500,
		BulkRPM:         50,
		ExportRPM:       100,
		BurstMultiplier: 1.5,
	},
	TierEnterprise: {
		DefaultRPM:      5000,
		ReadRPM:         10000,
		WriteRPM:        2500,
		BulkRPM:         200,
		ExportRPM:       500,
		BurstMultiplier: 2.0,
	},
}

// TierResolver looks up the subscription tier for a tenant.
// Implementations should cache results (e.g., Redis with 5-minute TTL) to avoid
// repeated lookups on every request.
type TierResolver interface {
	ResolveTier(ctx context.Context, tenantID uuid.UUID) (SubscriptionTier, error)
}

// RedisTierResolver resolves tenant tiers from a Redis cache.
// Falls back to TierProfessional when the tier is not cached.
// Tenants are expected to set their tier via:
//
//	SET tenant_tier:<id> = "enterprise"
//
// The resolved tier is cached in Redis with the configured TTL (default: 5 minutes).
type RedisTierResolver struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRedisTierResolver creates a RedisTierResolver with the given Redis client.
// The default TTL for cached tier lookups is 5 minutes.
func NewRedisTierResolver(rdb *redis.Client) *RedisTierResolver {
	return &RedisTierResolver{
		rdb: rdb,
		ttl: 5 * time.Minute,
	}
}

// ResolveTier looks up the subscription tier for the given tenant from Redis.
// Returns TierProfessional as a safe default when:
//   - The key does not exist in Redis (tenant not yet configured).
//   - Redis is unavailable (fail-open — never block legitimate requests due to infra issues).
//   - The stored tier value is not a recognized SubscriptionTier.
func (r *RedisTierResolver) ResolveTier(ctx context.Context, tenantID uuid.UUID) (SubscriptionTier, error) {
	if r.rdb == nil {
		return TierProfessional, nil
	}

	key := fmt.Sprintf("tenant_tier:%s", tenantID.String())
	val, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// Key not found — default to professional tier.
		return TierProfessional, nil
	}
	if err != nil {
		// Redis error — fail open with professional tier.
		return TierProfessional, nil
	}

	tier := SubscriptionTier(val)
	switch tier {
	case TierFree, TierProfessional, TierEnterprise:
		return tier, nil
	default:
		// Unknown tier value — default to professional tier.
		return TierProfessional, nil
	}
}

// ToGroupLimit converts tier limits to a GroupLimit for the given endpoint group.
// The mapping is:
//   - EndpointGroupRead  → ReadRPM
//   - EndpointGroupWrite → WriteRPM
//   - EndpointGroupUpload → BulkRPM
//   - All others          → DefaultRPM
//
// BurstPerSecond is calculated as: int(RPM * BurstMultiplier / 60), minimum 1.
func (tl TenantRateLimits) ToGroupLimit(group gwconfig.EndpointGroup) GroupLimit {
	var rpm int
	switch group {
	case gwconfig.EndpointGroupRead:
		rpm = tl.ReadRPM
	case gwconfig.EndpointGroupWrite:
		rpm = tl.WriteRPM
	case gwconfig.EndpointGroupUpload:
		rpm = tl.BulkRPM
	default:
		rpm = tl.DefaultRPM
	}

	burstPerSecond := int(float64(rpm) * tl.BurstMultiplier / 60.0)
	if burstPerSecond < 1 {
		burstPerSecond = 1
	}

	return GroupLimit{
		RequestsPerWindow: rpm,
		Window:            time.Minute,
		BurstPerSecond:    burstPerSecond,
	}
}
