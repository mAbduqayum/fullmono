// Command api serves the URL shortener: its JSON API and its redirects.
//
//	api              serve on $PORT (8080)
//	api migrate      apply the database migrations, then exit
//	api healthcheck  exit 0 if the server answers /api/health
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mAbduqayum/fullmono/apps/api/internal/links"
	"github.com/mAbduqayum/fullmono/apps/api/migrations"
)

// shutdownTimeout stays under the 10 s `docker stop` waits before SIGKILL.
const shutdownTimeout = 5 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("exiting", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if len(args) == 0 {
		return serve(port, databaseURL)
	}
	switch args[0] {
	case "migrate":
		return migrations.Up(databaseURL)
	case "healthcheck":
		return healthcheck(port)
	default:
		return fmt.Errorf("unknown command %q: want migrate or healthcheck", args[0])
	}
}

func serve(port, databaseURL string) error {
	// Ctrl+C, or the SIGTERM `docker stop` sends, starts a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           links.NewHandler(links.NewPostgresStore(pool)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	failed := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}
	slog.Info("shutting down")
	// Let the requests in flight finish, but no longer than shutdownTimeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// healthcheck is the image's HEALTHCHECK: distroless has no shell or curl to run one.
func healthcheck(port string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:"+port+"/api/health", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = res.Body.Close() // only the status matters
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("health check: %s", res.Status)
	}
	return nil
}
