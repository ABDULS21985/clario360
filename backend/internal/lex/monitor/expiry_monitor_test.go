package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/model"
)

func TestExpiry_90Days(t *testing.T) {
	now := time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
	contract := testExpiringContract("Contract 90", 85, false)
	var horizons []int

	monitor := &ExpiryMonitor{
		now:    func() time.Time { return now },
		logger: zerolog.Nop(),
		listDueFunc: func(_ context.Context, horizon int) ([]model.Contract, error) {
			if horizon == 90 {
				return []model.Contract{contract}, nil
			}
			return nil, nil
		},
		listExpiredFunc: func(context.Context) ([]model.Contract, error) { return nil, nil },
		notifyFunc: func(_ context.Context, _ *model.Contract, horizon int) error {
			horizons = append(horizons, horizon)
			return nil
		},
		expireFunc:    func(context.Context, *model.Contract) error { return nil },
		autoRenewFunc: func(context.Context, *model.Contract) error { return nil },
	}

	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(horizons) != 1 || horizons[0] != 90 {
		t.Fatalf("notified horizons = %v, want [90]", horizons)
	}
}

func TestExpiry_NoDuplicate(t *testing.T) {
	now := time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
	contract := testExpiringContract("Duplicate Contract", 25, false)
	seen := map[string]struct{}{}
	callCount := 0

	monitor := &ExpiryMonitor{
		now:    func() time.Time { return now },
		logger: zerolog.Nop(),
		listDueFunc: func(_ context.Context, horizon int) ([]model.Contract, error) {
			if horizon == 30 {
				return []model.Contract{contract}, nil
			}
			return nil, nil
		},
		listExpiredFunc: func(context.Context) ([]model.Contract, error) { return nil, nil },
		notifyFunc: func(_ context.Context, contract *model.Contract, horizon int) error {
			key := fmt.Sprintf("%s:%d", contract.ID, horizon)
			if _, exists := seen[key]; exists {
				return nil
			}
			seen[key] = struct{}{}
			callCount++
			return nil
		},
		expireFunc:    func(context.Context, *model.Contract) error { return nil },
		autoRenewFunc: func(context.Context, *model.Contract) error { return nil },
	}

	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if callCount != 1 {
		t.Fatalf("callCount = %d, want 1", callCount)
	}
}

func TestExpiry_AutoRenew(t *testing.T) {
	now := time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
	contract := testExpiringContract("Auto Renew Contract", -1, true)
	renewalDate := now.AddDate(0, 0, -2)
	contract.RenewalDate = &renewalDate
	autoRenewed := 0
	expired := 0

	monitor := &ExpiryMonitor{
		now:             func() time.Time { return now },
		logger:          zerolog.Nop(),
		listDueFunc:     func(context.Context, int) ([]model.Contract, error) { return nil, nil },
		listExpiredFunc: func(context.Context) ([]model.Contract, error) { return []model.Contract{contract}, nil },
		notifyFunc:      func(context.Context, *model.Contract, int) error { return nil },
		expireFunc: func(context.Context, *model.Contract) error {
			expired++
			return nil
		},
		autoRenewFunc: func(context.Context, *model.Contract) error {
			autoRenewed++
			return nil
		},
	}

	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if autoRenewed != 1 || expired != 0 {
		t.Fatalf("autoRenewed=%d expired=%d, want 1 and 0", autoRenewed, expired)
	}
}

func TestExpiry_Expired(t *testing.T) {
	now := time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
	contract := testExpiringContract("Expired Contract", -1, false)
	autoRenewed := 0
	expired := 0

	monitor := &ExpiryMonitor{
		now:             func() time.Time { return now },
		logger:          zerolog.Nop(),
		listDueFunc:     func(context.Context, int) ([]model.Contract, error) { return nil, nil },
		listExpiredFunc: func(context.Context) ([]model.Contract, error) { return []model.Contract{contract}, nil },
		notifyFunc:      func(context.Context, *model.Contract, int) error { return nil },
		expireFunc: func(context.Context, *model.Contract) error {
			expired++
			return nil
		},
		autoRenewFunc: func(context.Context, *model.Contract) error {
			autoRenewed++
			return nil
		},
	}

	if err := monitor.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if autoRenewed != 0 || expired != 1 {
		t.Fatalf("autoRenewed=%d expired=%d, want 0 and 1", autoRenewed, expired)
	}
}

func testExpiringContract(title string, expiryOffset int, autoRenew bool) model.Contract {
	expiryDate := time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC).AddDate(0, 0, expiryOffset)
	return model.Contract{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		Title:             title,
		PartyBName:        "Counterparty",
		ExpiryDate:        &expiryDate,
		AutoRenew:         autoRenew,
		RenewalNoticeDays: 30,
		Status:            model.ContractStatusActive,
		OwnerName:         "Owner User",
		Type:              model.ContractTypeServiceAgreement,
	}
}
