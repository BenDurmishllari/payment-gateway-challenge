package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/handlers"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"
)

const (
	bankSimulatorTimeout = 5 * time.Second
	shutdownTimeout      = 10 * time.Second
)

type Api struct {
	router          *chi.Mux
	paymentsHandler *handlers.PaymentsHandler
	paymentsRepo    *repository.PaymentsRepository
	authorizer      bank.Authorizer
	logger          *slog.Logger
}

func New(logger *slog.Logger) *Api {
	a := &Api{}
	a.paymentsRepo = repository.NewPaymentsRepository()
	a.authorizer = bank.NewClient(acquiringBankURL(), bankSimulatorTimeout)
	a.paymentsHandler = handlers.NewPaymentsHandler(a.paymentsRepo, a.authorizer, logger)
	a.logger = logger
	a.setupRouter()

	return a
}

func acquiringBankURL() string {
	if url := os.Getenv("BANK_SIMULATOR_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func (a *Api) Run(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: a.router,
	}

	g, runCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-runCtx.Done()
		slog.InfoContext(ctx, "shutting down HTTP server")

		// A fresh context, not runCtx: runCtx is already cancelled by this
		// point, and Shutdown treats an already-done context as "stop
		// immediately" rather than "wait for in-flight requests to drain."
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return httpServer.Shutdown(shutdownCtx)
	})

	g.Go(func() error {
		slog.InfoContext(ctx, "starting HTTP server", slog.String("addr", addr))
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	return g.Wait()
}

func (a *Api) setupRouter() {
	a.router = chi.NewRouter()
	a.router.Use(middleware.Logger)

	a.router.Get("/ping", a.PingHandler())
	a.router.Get("/swagger/*", a.SwaggerHandler())

	a.router.Get("/api/payments/{id}", a.paymentsHandler.GetHandler())
	a.router.Post("/api/payments", a.paymentsHandler.PostHandler())
}
