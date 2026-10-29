package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoAdmin means no admin has the username.
var ErrNoAdmin = errors.New("no such admin")

// Admins is what a login needs from the admin accounts.
type Admins interface {
	// PasswordHash returns the admin's bcrypt hash, or ErrNoAdmin.
	PasswordHash(ctx context.Context, username string) ([]byte, error)
}

// PostgresAdmins keeps the admins in the admins table.
type PostgresAdmins struct {
	pool *pgxpool.Pool
}

func NewPostgresAdmins(pool *pgxpool.Pool) *PostgresAdmins {
	return &PostgresAdmins{pool: pool}
}

func (a *PostgresAdmins) PasswordHash(ctx context.Context, username string) ([]byte, error) {
	var hash []byte
	err := a.pool.QueryRow(ctx,
		`SELECT password_hash FROM admins WHERE username = $1`, username,
	).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoAdmin
	}
	if err != nil {
		return nil, fmt.Errorf("find admin: %w", err)
	}
	return hash, nil
}

// Set creates the admin, or changes their password if they exist.
func (a *PostgresAdmins) Set(ctx context.Context, username string, hash []byte) error {
	_, err := a.pool.Exec(ctx,
		`INSERT INTO admins (username, password_hash) VALUES ($1, $2)
		 ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		username, hash,
	)
	if err != nil {
		return fmt.Errorf("set admin: %w", err)
	}
	return nil
}
