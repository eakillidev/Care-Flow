package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/users"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	users  users.Repository
	tokens *TokenManager
}

func NewService(userRepository users.Repository, tokens *TokenManager) *Service {
	return &Service{users: userRepository, tokens: tokens}
}

func (service *Service) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := service.users.GetByEmail(ctx, users.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("look up login user: %w", err)
	}
	if !VerifyPassword(user.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	token, err := service.tokens.Issue(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("issue login token: %w", err)
	}
	return &LoginResponse{Token: token, User: NewUserResponse(user)}, nil
}
