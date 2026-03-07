package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
)

type ClauseService struct {
	contracts *repository.ContractRepository
	clauses   *repository.ClauseRepository
	publisher Publisher
	metrics   *metrics.Metrics
	topic     string
	logger    zerolog.Logger
	now       func() time.Time
}

func NewClauseService(contracts *repository.ContractRepository, clauses *repository.ClauseRepository, publisher Publisher, appMetrics *metrics.Metrics, topic string, logger zerolog.Logger) *ClauseService {
	return &ClauseService{
		contracts: contracts,
		clauses:   clauses,
		publisher: publisherOrNoop(publisher),
		metrics:   appMetrics,
		topic:     topic,
		logger:    logger.With().Str("service", "lex-clauses").Logger(),
		now:       time.Now,
	}
}

func (s *ClauseService) List(ctx context.Context, tenantID, contractID uuid.UUID) ([]model.Clause, error) {
	if _, err := s.contracts.Get(ctx, tenantID, contractID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	return s.clauses.ListByContract(ctx, tenantID, contractID)
}

func (s *ClauseService) Get(ctx context.Context, tenantID, contractID, clauseID uuid.UUID) (*model.Clause, error) {
	clause, err := s.clauses.Get(ctx, tenantID, contractID, clauseID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("clause not found")
		}
		return nil, internalError("get clause", err)
	}
	return clause, nil
}

func (s *ClauseService) UpdateReview(ctx context.Context, tenantID, contractID, clauseID, reviewedBy uuid.UUID, req dto.UpdateClauseReviewRequest) (*model.Clause, error) {
	req.Normalize()
	now := s.now().UTC()
	if err := s.clauses.UpdateReview(ctx, s.clausesDB(), tenantID, contractID, clauseID, req.Status, &reviewedBy, req.Notes, now); err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("clause not found")
		}
		return nil, internalError("update clause review", err)
	}
	clause, err := s.clauses.Get(ctx, tenantID, contractID, clauseID)
	if err != nil {
		return nil, internalError("reload clause", err)
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.clause.reviewed", tenantID, &reviewedBy, map[string]any{
		"id":            clause.ID,
		"contract_id":   clause.ContractID,
		"review_status": clause.ReviewStatus,
		"reviewed_by":   reviewedBy,
	}, s.logger)
	return clause, nil
}

func (s *ClauseService) RiskSummary(ctx context.Context, tenantID, contractID uuid.UUID) ([]model.Clause, error) {
	return s.clauses.RiskSummary(ctx, tenantID, contractID)
}

func (s *ClauseService) clausesDB() repository.Queryer {
	return s.clausesDBPool()
}

func (s *ClauseService) clausesDBPool() repository.Queryer {
	return s.contractsDBPool()
}

func (s *ClauseService) contractsDBPool() repository.Queryer {
	return s.contractsDB()
}

func (s *ClauseService) contractsDB() repository.Queryer {
	return s.clausesDBInternal()
}

func (s *ClauseService) clausesDBInternal() repository.Queryer {
	return s.clausesDBAccessor()
}

func (s *ClauseService) clausesDBAccessor() repository.Queryer {
	return s.clausesDBPoolAccessor()
}

func (s *ClauseService) clausesDBPoolAccessor() repository.Queryer {
	return s.clausesDBBase()
}

func (s *ClauseService) clausesDBBase() repository.Queryer {
	return s.clauses.DB()
}
