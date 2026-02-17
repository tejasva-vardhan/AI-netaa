package service

import (
	"finalneta/repository"
	"fmt"
)

// UserService handles business logic for users
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetOrCreateUserByPhone gets existing user or creates new one
// Returns user_id and whether user was just created
func (s *UserService) GetOrCreateUserByPhone(phoneNumber string) (int64, bool, error) {
	// Check if user exists
	user, err := s.userRepo.GetUserByPhone(phoneNumber)
	if err != nil {
		return 0, false, fmt.Errorf("failed to check user existence: %w", err)
	}

	if user != nil {
		// User exists, return existing user_id
		return user.UserID, false, nil
	}

	// User doesn't exist, create new user
	userID, err := s.userRepo.CreateUser(phoneNumber)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create user: %w", err)
	}

	return userID, true, nil
}

// VerifyUserExists checks if user exists in database
func (s *UserService) VerifyUserExists(userID int64) (bool, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to verify user existence: %w", err)
	}

	return user != nil, nil
}

// VerifyUserPhoneVerified checks if user's phone is verified
func (s *UserService) VerifyUserPhoneVerified(userID int64) (bool, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to check phone verification: %w", err)
	}

	if user == nil {
		return false, nil
	}

	return user.PhoneVerifiedAt != nil, nil
}

// MarkPhoneVerified marks user's phone as verified
func (s *UserService) MarkPhoneVerified(userID int64) error {
	return s.userRepo.VerifyUserPhone(userID)
}
