// Command api serves the URL shortener: its JSON API and its redirects.
//
//	api                  serve on $PORT (8080)
//	api migrate          apply the database migrations, then exit
//	api healthcheck      exit 0 if the server answers /api/health
//	api set-admin <name> create an admin, or change their password: the first line on stdin
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mAbduqayum/fullmono/apps/api/internal/auth"
	"github.com/mAbduqayum/fullmono/apps/api/internal/httpx"
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
		return serve(port, databaseURL, os.Getenv("JWT_SECRET"))
	}
	switch args[0] {
	case "migrate":
		return migrations.Up(databaseURL)
	case "healthcheck":
		return healthcheck(port)
	case "set-admin":
		if len(args) != 2 {
			return errors.New("usage: api set-admin <username>, with the password on stdin")
		}
		return setAdmin(databaseURL, args[1], os.Stdin)
	default:
		return fmt.Errorf("unknown command %q: want migrate, healthcheck or set-admin", args[0])
	}
}

func serve(port, databaseURL, jwtSecret string) error {
	tokens, err := auth.NewTokens(jwtSecret)
	if err != nil {
		return err
	}

	// Ctrl+C, or the SIGTERM `docker stop` sends, starts a graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.Handle("/api/admin/session", auth.NewHandler(auth.NewPostgresAdmins(pool), tokens))
	mux.Handle("/", links.NewHandler(links.NewPostgresStore(pool)))
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpx.LogRequests(mux),
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

// setAdmin reads the password from stdin, not from an argument, so it never
// shows in the process list or the shell's history.
func setAdmin(databaseURL, username string, stdin io.Reader) error {
	if err := auth.CheckUsername(username); err != nil {
		return err
	}
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("read the password: %w", err)
	}
	hash, err := auth.HashPassword(strings.TrimRight(line, "\r\n"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()
	if err := auth.NewPostgresAdmins(pool).Set(ctx, username, hash); err != nil {
		return err
	}
	slog.Info("admin set", "username", username)
	return nil
}
