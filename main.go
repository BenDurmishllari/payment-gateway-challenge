package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/docs"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/api"
	"golang.org/x/sync/errgroup"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const forceExitTimeout = 15 * time.Second

// @title Payment Gateway Challenge Go
// @description Interview challenge for building a Payment Gateway - Go version
// @host localhost:8090
// @BasePath /
// @securityDefinitions.basic BasicAuth
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("starting application",
		slog.String("version", version),
		slog.String("commit", commit),
		slog.String("build_date", date),
	)

	docs.SwaggerInfo.Version = version

	if err := run(logger); err != nil {
		slog.Error("application exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a := api.New(logger)

	g, runCtx := errgroup.WithContext(ctx)

	// Register all long-running background workers. Additional workers
	// (e.g., message consumers) should be added as separate g.Go calls.
	g.Go(func() error {
		return a.Run(runCtx, ":8090")
	})

	// Wait for OS signal cancellation or a worker failure.
	<-runCtx.Done()

	return waitWithTimeout(context.Background(), logger, g, forceExitTimeout)
}

// waitWithTimeout blocks until every worker in g has returned, or force-exits
// the process if they take longer than timeout to drain. This guards against
// a goroutine that ignores context cancellation and would otherwise hang the
// process forever on shutdown.
func waitWithTimeout(ctx context.Context, logger *slog.Logger, g *errgroup.Group, timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- g.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		logger.ErrorContext(ctx, "workers failed to shut down within timeout, forcing exit",
			slog.Duration("timeout", timeout),
		)
		os.Exit(1)
		return nil
	}
}
