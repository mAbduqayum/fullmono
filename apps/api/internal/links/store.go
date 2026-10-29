// Package links shortens URLs: it stores them under short codes and
// redirects a code to its URL, counting the visits.
package links

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// Stats sums the links up, for the admin panel.
type Stats struct {
	Links  int64  `json:"links"`
	Clicks int64  `json:"clicks"`
	Top    []Link `json:"top"`
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

	// Search returns a page of the links whose code or URL holds query (every
	// link for ""), newest first, and how many match in all.
	Search(ctx context.Context, query string, limit, offset int) ([]Link, int, error)
	// Delete removes the code's link, or returns ErrNotFound.
	Delete(ctx context.Context, code string) error
	// Stats counts the links and their clicks, with the top most clicked.
	Stats(ctx context.Context, top int) (Stats, error)
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
	return collectLinks(rows)
}

func collectLinks(rows pgx.Rows) ([]Link, error) {
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

// likeEscaper makes a search's %, _ and \ match themselves, not stand for
// any text (%) or any one character (_) in an ILIKE pattern.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Search reads every row: a pattern that starts with % can't use a b-tree
// index. That's fine for an admin page; at scale, pg_trgm's indexes can.
func (s *PostgresStore) Search(ctx context.Context, query string, limit, offset int) ([]Link, int, error) {
	pattern := "%" + likeEscaper.Replace(query) + "%"
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM links WHERE code ILIKE $1 OR url ILIKE $1`, pattern,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count links: %w", err)
	}
	// id breaks ties between links made in the same instant, so pages never overlap.
	rows, err := s.pool.Query(ctx,
		`SELECT code, url, clicks, created_at FROM links
		 WHERE code ILIKE $1 OR url ILIKE $1
		 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("search links: %w", err)
	}
	links, err := collectLinks(rows)
	return links, total, err
}

func (s *PostgresStore) Delete(ctx context.Context, code string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM links WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Stats(ctx context.Context, top int) (Stats, error) {
	var stats Stats
	// sum() of bigints is a numeric, and NULL over no rows at all.
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*), coalesce(sum(clicks), 0)::bigint FROM links`,
	).Scan(&stats.Links, &stats.Clicks); err != nil {
		return Stats{}, fmt.Errorf("count links: %w", err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT code, url, clicks, created_at FROM links WHERE clicks > 0
		 ORDER BY clicks DESC, created_at DESC LIMIT $1`,
		top,
	)
	if err != nil {
		return Stats{}, fmt.Errorf("top links: %w", err)
	}
	stats.Top, err = collectLinks(rows)
	return stats, err
}
