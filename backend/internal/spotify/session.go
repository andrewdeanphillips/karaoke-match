package spotify

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// session is one visitor's persisted Spotify OAuth session: their access and
// refresh tokens, and when the access token expires. ID is the same opaque
// value used as the session cookie, so a lookup by cookie value is a lookup
// by primary key.
type session struct {
	ID           string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// sessionStore is the subset of postgresSessionStore that Client depends on —
// narrow enough that tests can substitute a fake and exercise session/refresh
// logic without making real database queries.
type sessionStore interface {
	create(ctx context.Context, sess session) error
	lookup(ctx context.Context, id string) (session, bool, error)
	updateTokens(ctx context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error
}

type postgresSessionStore struct {
	pool *pgxpool.Pool
}

func newPostgresSessionStore(pool *pgxpool.Pool) *postgresSessionStore {
	return &postgresSessionStore{pool: pool}
}

func (s *postgresSessionStore) create(ctx context.Context, sess session) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO spotify_sessions (id, access_token, refresh_token, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		sess.ID, sess.AccessToken, sess.RefreshToken, sess.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("creating session %q: %w", sess.ID, err)
	}
	return nil
}

func (s *postgresSessionStore) lookup(ctx context.Context, id string) (session, bool, error) {
	sess := session{ID: id}

	err := s.pool.QueryRow(ctx,
		`SELECT access_token, refresh_token, expires_at
		 FROM spotify_sessions
		 WHERE id = $1`,
		id,
	).Scan(&sess.AccessToken, &sess.RefreshToken, &sess.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return session{}, false, nil
	}
	if err != nil {
		return session{}, false, fmt.Errorf("looking up session %q: %w", id, err)
	}

	return sess, true, nil
}

// updateTokens overwrites a session's tokens and expiry after a refresh —
// Spotify doesn't always issue a new refresh token on renewal, so callers
// pass through the existing one when it omits a replacement.
func (s *postgresSessionStore) updateTokens(ctx context.Context, id, accessToken, refreshToken string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE spotify_sessions
		 SET access_token = $2, refresh_token = $3, expires_at = $4
		 WHERE id = $1`,
		id, accessToken, refreshToken, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("updating session %q: %w", id, err)
	}
	return nil
}
