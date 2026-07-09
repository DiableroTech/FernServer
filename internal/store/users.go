package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email already registered")

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     string
	DefaultModality string
	CreatedAt       time.Time
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	u := &User{Email: email, PasswordHash: passwordHash, DisplayName: displayName}
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, display_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		email, passwordHash, displayName,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, default_modality, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.DefaultModality, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, display_name, default_modality, created_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.DefaultModality, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id, displayName, defaultModality string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET display_name = $2, default_modality = $3, updated_at = now() WHERE id = $1`,
		id, displayName, defaultModality,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// DeleteUser hard-deletes the account; all journal data cascades.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
