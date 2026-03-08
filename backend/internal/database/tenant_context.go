package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetTenantContext sets the PostgreSQL session variable that Row-Level Security policies check.
//
// It executes: SET LOCAL app.current_tenant_id = '<uuid>'
//
// SET LOCAL scopes the variable to the current transaction block only. When the transaction
// ends (commit or rollback), the variable is automatically reset and cannot leak to other
// connections or transactions in the pool.
//
// RLS policies are expected to reference this variable via:
//
//	current_setting('app.current_tenant_id', true)::uuid
//
// The second argument (true = missing_ok) means that if app.current_tenant_id is not set,
// current_setting returns NULL instead of raising an error. NULL ::uuid cast also returns NULL,
// and NULL = anything is always FALSE in SQL, so no rows are visible — the correct safe default
// when no tenant context has been established.
func SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context (tenant=%s): %w", tenantID, err)
	}
	return nil
}

// RunWithTenant executes fn within a transaction where app.current_tenant_id is set to tenantID.
//
// The tenant context is strictly scoped to this transaction via SET LOCAL — it cannot leak to
// other connections in the pool once the transaction ends. If fn returns an error, the transaction
// is rolled back. If fn returns nil, the transaction is committed.
//
// This is the primary helper for all write operations in tenant-scoped services. RLS policies
// on every tenant-scoped table will automatically filter rows to those belonging to tenantID.
//
// Example:
//
//	err := database.RunWithTenant(ctx, pool, tenantID, func(tx pgx.Tx) error {
//	    _, err := tx.Exec(ctx, "INSERT INTO assets (tenant_id, name) VALUES ($1, $2)", tenantID, name)
//	    return err
//	})
func RunWithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	//nolint:errcheck
	defer tx.Rollback(ctx)

	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RunReadWithTenant is like RunWithTenant but opens a read-only transaction.
//
// Use this for SELECT queries where you want RLS to filter rows by tenant but do not need
// to make any changes. The read-only transaction mode prevents accidental writes and may
// allow the database to use read replicas if the connection pool is configured accordingly.
//
// The app.current_tenant_id session variable is set via SET LOCAL, scoped to this transaction,
// and is automatically cleared when the transaction ends.
func RunReadWithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("begin read-only transaction: %w", err)
	}
	//nolint:errcheck
	defer tx.Rollback(ctx)

	if err := SetTenantContext(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	// Commit is still called for read-only transactions to cleanly terminate them.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit read-only transaction: %w", err)
	}

	return nil
}
