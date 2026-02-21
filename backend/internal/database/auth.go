package database

import (
	"context"
	"errors"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/models"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserExists        = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound   = errors.New("session not found")
)

// CreateUser creates a new user with the given email and password hash
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}

	err := db.Pool.QueryRow(
		ctx,
		"INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3) RETURNING created_at",
		user.ID, user.Email, user.PasswordHash,
	).Scan(&user.CreatedAt)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, email, password_hash, created_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, email, password_hash, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if isNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// CreateSession creates a new session for a user
func (db *DB) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	err := db.Pool.QueryRow(
		ctx,
		"INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3) RETURNING created_at",
		session.ID, session.UserID, session.ExpiresAt,
	).Scan(&session.CreatedAt)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by ID, returns error if expired or not found
func (db *DB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	session := &models.Session{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1 AND expires_at > NOW()",
		id,
	).Scan(&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt)

	if err != nil {
		if isNotFound(err) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes a session by ID
func (db *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	// PostgreSQL unique violation error code: 23505
	return err != nil && err.Error() != "" && 
		(err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" ||
		 err.Error() == "ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)")
}

// isNotFound checks if the error is a no rows error
func isNotFound(err error) bool {
	return err != nil && err.Error() == "no rows in result set"
}
