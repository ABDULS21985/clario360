package consumer

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
)

type ActaConsumer struct {
	store    *repository.Store
	consumer *events.Consumer
	logger   zerolog.Logger
}

func NewActaConsumer(store *repository.Store, consumer *events.Consumer, logger zerolog.Logger) *ActaConsumer {
	handler := &ActaConsumer{
		store:    store,
		consumer: consumer,
		logger:   logger.With().Str("component", "acta_consumer").Logger(),
	}
	if consumer != nil {
		consumer.Subscribe(events.Topics.IAMEvents, handler)
	}
	return handler
}

func (c *ActaConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.user.deleted", "com.clario360.iam.user.deleted":
		return c.handleUserDeleted(ctx, event)
	default:
		return nil
	}
}

func (c *ActaConsumer) Start(ctx context.Context) error {
	if c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *ActaConsumer) Stop() error {
	if c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *ActaConsumer) handleUserDeleted(ctx context.Context, event *events.Event) error {
	if strings.TrimSpace(event.TenantID) == "" || strings.TrimSpace(event.UserID) == "" {
		return nil
	}
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return nil
	}
	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		return nil
	}
	now := time.Now().UTC()
	var memberships, leadershipAssignments, deferredActionItems int64
	if err := database.RunInTx(ctx, c.store.DB(), func(tx pgx.Tx) error {
		var err error
		memberships, err = c.store.DeactivateMembershipsByUser(ctx, tx, tenantID, userID, now)
		if err != nil {
			return err
		}
		leadershipAssignments, err = c.store.ClearLeadershipAssignmentsByUser(ctx, tx, tenantID, userID, now)
		if err != nil {
			return err
		}
		deferredActionItems, err = c.store.DeferActionItemsAssignedToUser(ctx, tx, tenantID, userID, "assignee removed from IAM", now)
		return err
	}); err != nil {
		return err
	}
	c.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("user_id", userID.String()).
		Int64("memberships_deactivated", memberships).
		Int64("leadership_assignments_cleared", leadershipAssignments).
		Int64("action_items_deferred", deferredActionItems).
		Msg("processed IAM user deletion event for acta")
	return nil
}
