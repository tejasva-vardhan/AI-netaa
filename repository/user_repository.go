package repository

import (
	"database/sql"
	"fmt"
	"time"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserByPhone retrieves user by phone number
func (r *UserRepository) GetUserByPhone(phoneNumber string) (*User, error) {
	query := `
		SELECT user_id, phone_number, phone_verified_at, created_at
		FROM users
		WHERE phone_number = ?
		LIMIT 1
	`

	user := &User{}
	var verifiedAt sql.NullTime
	err := r.db.QueryRow(query, phoneNumber).Scan(
		&user.UserID,
		&user.PhoneNumber,
		&verifiedAt,
		&user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // User doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	if verifiedAt.Valid {
		user.PhoneVerifiedAt = &verifiedAt.Time
	}

	return user, nil
}

// GetUserByID retrieves user by ID
func (r *UserRepository) GetUserByID(userID int64) (*User, error) {
	query := `
		SELECT user_id, phone_number, phone_verified_at, created_at
		FROM users
		WHERE user_id = ?
		LIMIT 1
	`

	user := &User{}
	var verifiedAt sql.NullTime
	err := r.db.QueryRow(query, userID).Scan(
		&user.UserID,
		&user.PhoneNumber,
		&verifiedAt,
		&user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // User doesn't exist
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	if verifiedAt.Valid {
		user.PhoneVerifiedAt = &verifiedAt.Time
	}

	return user, nil
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(phoneNumber string) (int64, error) {
	query := `
		INSERT INTO users (phone_number, created_at)
		VALUES (?, NOW())
	`

	result, err := r.db.Exec(query, phoneNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return userID, nil
}

// VerifyUserPhone marks user's phone as verified
func (r *UserRepository) VerifyUserPhone(userID int64) error {
	query := `
		UPDATE users
		SET phone_verified_at = NOW(),
		    last_active_at = NOW()
		WHERE user_id = ?
	`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to verify user phone: %w", err)
	}

	return nil
}

// UpdateLastActive updates user's last active timestamp
func (r *UserRepository) UpdateLastActive(userID int64) error {
	query := `
		UPDATE users
		SET last_active_at = NOW()
		WHERE user_id = ?
	`

	_, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last active: %w", err)
	}

	return nil
}

// User represents a user entity
type User struct {
	UserID          int64
	PhoneNumber     string
	PhoneVerifiedAt *time.Time
	CreatedAt       time.Time
}
