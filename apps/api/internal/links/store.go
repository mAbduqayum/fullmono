// Package links shortens URLs: it stores them under short codes and
// redirects a code to its URL, counting the visits.
package links

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Link is a shortened URL.
type Link struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	Clicks    int64     `json:"clicks"`
	CreatedAt time.Time `json:"createdAt"`
}

var (
	// ErrNotFound means no link has the code.
	ErrNotFound = errors.New("link not found")
	// ErrCodeTaken means another link already has the code.
	ErrCodeTaken = errors.New("code already taken")
)

// Store keeps the links.
type Store interface {
	Create(ctx context.Context, code, url string) (Link, error)
	// List returns the newest links first, at most limit of them.
	List(ctx context.Context, limit int) ([]Link, error)
	// Visit counts a visit to the code's link and returns its URL.
	Visit(ctx context.Context, code string) (string, error)
	Ping(ctx context.Context) error
}

// PostgresStore keeps the links in the links table.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// uniqueViolation is Postgres's SQLSTATE for a duplicate key.
const uniqueViolation = "23505"

func (s *PostgresStore) Create(ctx context.Context, code, url string) (Link, error) {
	var link Link
	err := s.pool.QueryRow(ctx,
		`INSERT INTO links (code, url) VALUES ($1, $2)
		 RETURNING code, url, clicks, created_at`,
		code, url,
	).Scan(&link.Code, &link.URL, &link.Clicks, &link.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return Link{}, ErrCodeTaken
	}
	if err != nil {
		return Link{}, fmt.Errorf("insert link: %w", err)
	}
	return link, nil
}

func (s *PostgresStore) List(ctx context.Context, limit int) ([]Link, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT code, url, clicks, created_at FROM links
		 ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Link, error) {
		var link Link
		err := row.Scan(&link.Code, &link.URL, &link.Clicks, &link.CreatedAt)
		return link, err
	})
}

func (s *PostgresStore) Visit(ctx context.Context, code string) (string, error) {
	var url string
	err := s.pool.QueryRow(ctx,
		`UPDATE links SET clicks = clicks + 1 WHERE code = $1 RETURNING url`,
		code,
	).Scan(&url)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("visit link: %w", err)
	}
	return url, nil
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
