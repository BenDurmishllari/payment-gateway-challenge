package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/sync/errgroup"
)

const bankSimulatorTimeout = 5 * time.Second

type Api struct {
	router       *chi.Mux
	paymentsRepo *repository.PaymentsRepository
	authorizer   bank.Authorizer
}

func New() *Api {
	a := &Api{}
	a.paymentsRepo = repository.NewPaymentsRepository()
	a.authorizer = bank.NewClient(bankSimulatorURL(), bankSimulatorTimeout)
	a.setupRouter()

	return a
}

// bankSimulatorURL returns the acquiring bank's base URL, defaulting to the
// bank simulator's address when running via docker-compose.
func bankSimulatorURL() string {
	if url := os.Getenv("BANK_SIMULATOR_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func (a *Api) Run(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:        addr,
		Handler:     a.router,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		<-ctx.Done()
		fmt.Printf("shutting down HTTP server\n")
		return httpServer.Shutdown(ctx)
	})

	g.Go(func() error {
		fmt.Printf("starting HTTP server on %s\n", addr)
		err := httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
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

	a.router.Get("/api/payments/{id}", a.GetPaymentHandler())
	a.router.Post("/api/payments", a.PostPaymentHandler())
}
