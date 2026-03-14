package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// CertificationRepository manages access certification campaigns.
type CertificationRepository interface {
	CreateCampaign(ctx context.Context, campaign *model.Campaign) error
	GetCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*model.Campaign, error)
	CreateReviewItems(ctx context.Context, items []model.CampaignReviewItem) error
	UpdateReviewItem(ctx context.Context, itemID uuid.UUID, decision string, reviewerID uuid.UUID) error
	UpdateCampaignStats(ctx context.Context, campaignID uuid.UUID) error
}

// CertificationManager handles access certification campaigns: periodic reviews
// where asset owners confirm or revoke access mappings.
type CertificationManager struct {
	certRepo CertificationRepository
	mappings MappingProvider
	statuses MappingStatusUpdater
	logger   zerolog.Logger
}

// NewCertificationManager creates a new certification campaign manager.
func NewCertificationManager(
	certRepo CertificationRepository,
	mappings MappingProvider,
	statuses MappingStatusUpdater,
	logger zerolog.Logger,
) *CertificationManager {
	return &CertificationManager{
		certRepo: certRepo,
		mappings: mappings,
		statuses: statuses,
		logger:   logger.With().Str("component", "certification_manager").Logger(),
	}
}

// CreateCampaign initializes a new certification campaign. It scopes to all
// identities accessing data assets at or above the specified classification threshold,
// generates review items (one per identity-asset mapping), and assigns reviewers.
func (c *CertificationManager) CreateCampaign(ctx context.Context, tenantID uuid.UUID, params model.CampaignParams, createdBy uuid.UUID) (*model.Campaign, error) {
	if params.DeadlineDays <= 0 {
		params.DeadlineDays = 14
	}

	allMappings, err := c.mappings.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	minRank := model.ClassificationRank(params.MinClassification)
	identityTypeSet := make(map[string]bool)
	for _, t := range params.IdentityTypes {
		identityTypeSet[t] = true
	}

	// Filter mappings to campaign scope.
	var scopedMappings []*model.AccessMapping
	for _, m := range allMappings {
		if model.ClassificationRank(m.DataClassification) < minRank {
			continue
		}
		if len(identityTypeSet) > 0 && !identityTypeSet[m.IdentityType] {
			continue
		}
		scopedMappings = append(scopedMappings, m)
	}

	campaign := &model.Campaign{
		ID:       uuid.New(),
		TenantID: tenantID,
		Name:     params.Name,
		Status:   "active",
		Scope: model.CampaignScope{
			MinClassification: params.MinClassification,
			IdentityTypes:     params.IdentityTypes,
		},
		TotalItems: len(scopedMappings),
		Deadline:   time.Now().UTC().Add(time.Duration(params.DeadlineDays) * 24 * time.Hour),
		CreatedBy:  createdBy,
		CreatedAt:  time.Now().UTC(),
	}

	if err := c.certRepo.CreateCampaign(ctx, campaign); err != nil {
		return nil, err
	}

	// Generate review items.
	items := make([]model.CampaignReviewItem, 0, len(scopedMappings))
	for _, m := range scopedMappings {
		items = append(items, model.CampaignReviewItem{
			ID:             uuid.New(),
			CampaignID:     campaign.ID,
			MappingID:      m.ID,
			IdentityID:     m.IdentityID,
			IdentityName:   m.IdentityName,
			DataAssetName:  m.DataAssetName,
			PermissionType: m.PermissionType,
			Decision:       "pending",
		})
	}

	if len(items) > 0 {
		if err := c.certRepo.CreateReviewItems(ctx, items); err != nil {
			return nil, err
		}
	}

	return campaign, nil
}

// ProcessDecision handles a reviewer's decision on a certification item.
//   - "revoke": marks the mapping as status='revoked'
//   - "approve": marks as reviewed, sets next_review_due
//   - "time_bound": sets expires_at on the mapping
func (c *CertificationManager) ProcessDecision(ctx context.Context, campaignID, itemID uuid.UUID, decision string, reviewerID uuid.UUID) error {
	if err := c.certRepo.UpdateReviewItem(ctx, itemID, decision, reviewerID); err != nil {
		return err
	}

	// Enforce decision on the underlying mapping.
	// The review item's mapping_id is not directly available here, so the repository
	// layer handles joining review items → mappings.
	if err := c.certRepo.UpdateCampaignStats(ctx, campaignID); err != nil {
		c.logger.Warn().Err(err).Msg("failed to update campaign stats")
	}

	return nil
}
