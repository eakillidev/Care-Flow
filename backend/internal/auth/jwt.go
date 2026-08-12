package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/users"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Identity struct {
	UserID uuid.UUID
	Role   users.Role
}

type Claims struct {
	Role users.Role `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret     []byte
	expiration time.Duration
	now        func() time.Time
}

func NewTokenManager(secret string, expiration time.Duration) (*TokenManager, error) {
	if secret == "" {
		return nil, errors.New("JWT secret is required")
	}
	if expiration <= 0 {
		return nil, errors.New("JWT expiration must be positive")
	}
	return &TokenManager{
		secret:     []byte(secret),
		expiration: expiration,
		now:        time.Now,
	}, nil
}

func (manager *TokenManager) Issue(userID uuid.UUID, role users.Role) (string, error) {
	now := manager.now().UTC()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.expiration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(manager.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}

func (manager *TokenManager) Validate(encoded string) (Identity, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		encoded,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return manager.secret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(manager.now),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return Identity{}, ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil || !validRole(claims.Role) {
		return Identity{}, ErrInvalidToken
	}
	return Identity{UserID: userID, Role: claims.Role}, nil
}

func validRole(role users.Role) bool {
	return role == users.RoleCaregiver || role == users.RoleCoordinator
}
