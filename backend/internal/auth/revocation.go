package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RevocationStore persiste tokens invalidados (logout) en la tabla revoked_tokens.
type RevocationStore struct {
	pool *pgxpool.Pool
}

func NewRevocationStore(pool *pgxpool.Pool) *RevocationStore {
	return &RevocationStore{pool: pool}
}

func (s *RevocationStore) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO revoked_tokens (jti, expires_at) VALUES ($1, $2)
		 ON CONFLICT (jti) DO NOTHING`,
		jti, expiresAt,
	)
	return err
}

func (s *RevocationStore) IsRevoked(ctx context.Context, jti string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE jti = $1)`,
		jti,
	).Scan(&exists)
	return exists, err
}
