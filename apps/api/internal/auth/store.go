package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mAbduqayum/fullmono/apps/api/internal/db"
)

// ErrNoAdmin means no admin has the username.
var ErrNoAdmin = errors.New("no such admin")

// Admins is what a login needs from the admin accounts.
type Admins interface {
	// PasswordHash returns the admin's bcrypt hash, or ErrNoAdmin.
	PasswordHash(ctx context.Context, username string) ([]byte, error)
}

// PostgresAdmins keeps the admins in the admins table, through the queries
// sqlc generates from internal/db/admins.sql.
type PostgresAdmins struct {
	queries *db.Queries
}

func NewPostgresAdmins(pool *pgxpool.Pool) *PostgresAdmins {
	return &PostgresAdmins{queries: db.New(pool)}
}

func (a *PostgresAdmins) PasswordHash(ctx context.Context, username string) ([]byte, error) {
	hash, err := a.queries.AdminPasswordHash(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoAdmin
	}
	if err != nil {
		return nil, fmt.Errorf("find admin: %w", err)
	}
	return []byte(hash), nil
}

// Set creates the admin, or changes their password if they exist.
func (a *PostgresAdmins) Set(ctx context.Context, username string, hash []byte) error {
	err := a.queries.SetAdmin(ctx, db.SetAdminParams{Username: username, PasswordHash: string(hash)})
	if err != nil {
		return fmt.Errorf("set admin: %w", err)
	}
	return nil
}
