// Package links shortens URLs: it stores them under short codes and
// redirects a code to its URL, counting the visits.
package links

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mAbduqayum/fullmono/apps/api/internal/db"
)

// Link and Stats, what the store returns, are generated from openapi.yaml's
// schemas (api.gen.go): the store fills in the very JSON the api answers.

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

// PostgresStore keeps the links in the links table, through the queries sqlc
// generates from internal/db/links.sql.
type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, queries: db.New(pool)}
}

// uniqueViolation is Postgres's SQLSTATE for a duplicate key.
const uniqueViolation = "23505"

func (s *PostgresStore) Create(ctx context.Context, code, url string) (Link, error) {
	row, err := s.queries.CreateLink(ctx, db.CreateLinkParams{Code: code, URL: url})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return Link{}, ErrCodeTaken
	}
	if err != nil {
		return Link{}, fmt.Errorf("insert link: %w", err)
	}
	return toLink(row), nil
}

func (s *PostgresStore) List(ctx context.Context, limit int) ([]Link, error) {
	rows, err := s.queries.ListLinks(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	return toLinks(rows), nil
}

// toLink is a row of the links table as the api answers it: without its id.
func toLink(row db.Link) Link {
	return Link{Code: row.Code, URL: row.URL, Clicks: row.Clicks, CreatedAt: row.CreatedAt}
}

// toLinks never returns nil, which would be JSON's null rather than [].
func toLinks(rows []db.Link) []Link {
	links := make([]Link, len(rows))
	for i, row := range rows {
		links[i] = toLink(row)
	}
	return links
}

func (s *PostgresStore) Visit(ctx context.Context, code string) (string, error) {
	url, err := s.queries.VisitLink(ctx, code)
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
	total, err := s.queries.CountMatchingLinks(ctx, pattern)
	if err != nil {
		return nil, 0, fmt.Errorf("count links: %w", err)
	}
	// openapi.yaml caps the page, which keeps the offset well inside an int32.
	rows, err := s.queries.SearchLinks(ctx, db.SearchLinksParams{
		Pattern: pattern, PageSize: int32(limit), Skip: int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("search links: %w", err)
	}
	return toLinks(rows), int(total), nil
}

func (s *PostgresStore) Delete(ctx context.Context, code string) error {
	deleted, err := s.queries.DeleteLink(ctx, code)
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) Stats(ctx context.Context, top int) (Stats, error) {
	counts, err := s.queries.CountLinksAndClicks(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("count links: %w", err)
	}
	rows, err := s.queries.TopLinks(ctx, int32(top))
	if err != nil {
		return Stats{}, fmt.Errorf("top links: %w", err)
	}
	return Stats{Links: counts.Links, Clicks: counts.Clicks, Top: toLinks(rows)}, nil
}
