package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/dashboard"
	"github.com/clario360/platform/internal/data/dto"
)

type DashboardService struct {
	calculator *dashboard.Calculator
}

func NewDashboardService(calculator *dashboard.Calculator) *DashboardService {
	return &DashboardService{calculator: calculator}
}

func (s *DashboardService) Get(ctx context.Context, tenantID uuid.UUID) (*dto.DataSuiteDashboard, error) {
	return s.calculator.Calculate(ctx, tenantID)
}
