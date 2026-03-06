package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// Run starts all HTTP servers and blocks until a shutdown signal is received.
//
// Startup:
//  1. Start admin HTTP server on AdminPort.
//  2. Start main HTTP server on Port.
//
// Shutdown order (on SIGINT or SIGTERM):
//  1. Main HTTP server (stop accepting new requests, drain in-flight).
//  2. Admin HTTP server.
//  3. Tracer flush (send remaining spans before connections close).
//  4. Database pool close (if initialized).
//  5. Redis client close (if initialized).
//
// Each shutdown step has its own timeout. If any step hangs beyond timeout,
// the error is logged and shutdown proceeds to the next step.
func (s *Service) Run(ctx context.Context) error {
	mainSrv := s.MainServer()
	adminSrv := s.AdminServer()

	g, gCtx := errgroup.WithContext(ctx)

	// Start admin server.
	g.Go(func() error {
		s.Logger.Info().Str("addr", adminSrv.Addr).Msg("admin server starting")
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("admin server: %w", err)
		}
		return nil
	})

	// Start main server.
	g.Go(func() error {
		s.Logger.Info().Str("addr", mainSrv.Addr).Msg("main server starting")
		if err := mainSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("main server: %w", err)
		}
		return nil
	})

	// Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		s.Logger.Info().Str("signal", sig.String()).Msg("shutdown initiated")
	case <-gCtx.Done():
		s.Logger.Info().Msg("shutdown initiated (context cancelled)")
	}

	// Ordered shutdown with per-step timeouts.
	timeout := s.Config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Step 1: Shutdown main HTTP server.
	s.shutdownStep("main server", timeout, func(ctx context.Context) error {
		return mainSrv.Shutdown(ctx)
	})

	// Step 2: Shutdown admin HTTP server.
	s.shutdownStep("admin server", 5*time.Second, func(ctx context.Context) error {
		return adminSrv.Shutdown(ctx)
	})

	// Step 3: Flush tracer.
	if s.tracerShutdown != nil {
		s.shutdownStep("tracer", 10*time.Second, s.tracerShutdown)
	}

	// Step 4: Close database pool.
	if s.DBPool != nil {
		s.shutdownStep("database", 5*time.Second, func(_ context.Context) error {
			s.DBPool.Close()
			return nil
		})
	}

	// Step 5: Close Redis.
	if s.Redis != nil {
		s.shutdownStep("redis", 5*time.Second, func(_ context.Context) error {
			return s.Redis.Close()
		})
	}

	s.Logger.Info().Msg("shutdown complete")

	// Wait for errgroup to finish (servers should have returned by now).
	_ = g.Wait()

	return nil
}

// shutdownStep runs a single shutdown step with a timeout.
// If the step hangs beyond timeout, it logs the hang and returns.
func (s *Service) shutdownStep(name string, timeout time.Duration, fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := fn(ctx); err != nil {
		s.Logger.Error().Err(err).Str("step", name).Msg("shutdown step failed")
	} else {
		s.Logger.Debug().Str("step", name).Msg("shutdown step completed")
	}
}
