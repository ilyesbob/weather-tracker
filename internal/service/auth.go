// Package service implements the application's business rules. It sits between
// the transport (handler) and infrastructure (repository) layers and depends
// only on small interfaces, which keeps it easy to test and to reuse.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository"
)

// AuthService handles registration, login and access-token verification.
type AuthService struct {
	users  repository.UserRepository
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewAuthService builds the authentication service.
func NewAuthService(users repository.UserRepository, secret string, ttl time.Duration) *AuthService {
	return &AuthService{
		users:  users,
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

// Register creates a new account and returns it with a signed token.
func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return domain.User{}, "", fmt.Errorf("%w: invalid email", domain.ErrValidation)
	}
	if len(password) < 8 {
		return domain.User{}, "", fmt.Errorf("%w: password must be at least 8 characters", domain.ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}

	user := domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    s.now().UTC(),
	}
	created, err := s.users.Create(ctx, user)
	if err != nil {
		return domain.User{}, "", err
	}

	token, err := s.issue(created.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return created, token, nil
}

// Login verifies credentials and returns a signed token.
func (s *AuthService) Login(ctx context.Context, email, password string) (domain.User, string, error) {
	user, err := s.users.ByEmail(ctx, email)
	if err != nil {
		// Return the same error shape for "not found" and "wrong password" to
		// avoid leaking which emails are registered.
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, "", fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
		}
		return domain.User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, "", fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	token, err := s.issue(user.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	return user, token, nil
}

// Verify parses a token and returns the subject (user ID).
func (s *AuthService) Verify(raw string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", domain.ErrUnauthorized, err)
	}
	if !token.Valid || claims.Subject == "" {
		return "", fmt.Errorf("%w: invalid token", domain.ErrUnauthorized)
	}
	return claims.Subject, nil
}

func (s *AuthService) issue(userID string) (string, error) {
	now := s.now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
