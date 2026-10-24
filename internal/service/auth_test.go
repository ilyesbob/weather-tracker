package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"weathertracker/internal/domain"
	"weathertracker/internal/repository/memory"
)

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{name: "happy path", email: "rider@example.com", password: "s3cret-passphrase"},
		{name: "invalid email", email: "not-an-email", password: "s3cret-passphrase", wantErr: domain.ErrValidation},
		{name: "empty email", email: "", password: "s3cret-passphrase", wantErr: domain.ErrValidation},
		{name: "short password", email: "rider@example.com", password: "short", wantErr: domain.ErrValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(memory.NewUserStore(), "test-secret", time.Hour)
			user, token, err := svc.Register(context.Background(), tt.email, tt.password)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() unexpected error: %v", err)
			}
			if user.ID == "" || token == "" {
				t.Fatalf("Register() returned empty user ID or token: %+v", user)
			}
			if user.PasswordHash == tt.password {
				t.Fatal("password must not be stored in plain text")
			}
		})
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc := NewAuthService(memory.NewUserStore(), "test-secret", time.Hour)
	ctx := context.Background()

	if _, _, err := svc.Register(ctx, "rider@example.com", "s3cret-passphrase"); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if _, _, err := svc.Register(ctx, "RIDER@example.com", "s3cret-passphrase"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected ErrConflict for duplicate email, got %v", err)
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{name: "happy path", email: "rider@example.com", password: "s3cret-passphrase"},
		{name: "wrong password", email: "rider@example.com", password: "wrong-password", wantErr: domain.ErrUnauthorized},
		{name: "unknown user", email: "ghost@example.com", password: "s3cret-passphrase", wantErr: domain.ErrUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(memory.NewUserStore(), "test-secret", time.Hour)
			if _, _, err := svc.Register(ctx, "rider@example.com", "s3cret-passphrase"); err != nil {
				t.Fatalf("seed register failed: %v", err)
			}

			user, token, err := svc.Login(ctx, tt.email, tt.password)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Login() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Login() unexpected error: %v", err)
			}
			if user.Email != tt.email || token == "" {
				t.Fatalf("Login() = %+v, token=%q", user, token)
			}
		})
	}
}

func TestAuthService_Verify(t *testing.T) {
	svc := NewAuthService(memory.NewUserStore(), "test-secret", time.Hour)
	ctx := context.Background()

	_, token, err := svc.Register(ctx, "rider@example.com", "s3cret-passphrase")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	t.Run("valid token returns subject", func(t *testing.T) {
		subject, err := svc.Verify(token)
		if err != nil {
			t.Fatalf("Verify() error: %v", err)
		}
		if subject == "" {
			t.Fatal("expected non-empty subject")
		}
	})

	t.Run("garbage token", func(t *testing.T) {
		if _, err := svc.Verify("not-a-jwt"); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("token signed with another secret", func(t *testing.T) {
		other := NewAuthService(memory.NewUserStore(), "different-secret", time.Hour)
		_, otherToken, err := other.Register(ctx, "other@example.com", "s3cret-passphrase")
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
		if _, err := svc.Verify(otherToken); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		expired := NewAuthService(memory.NewUserStore(), "test-secret", -time.Minute)
		_, expiredToken, err := expired.Register(ctx, "old@example.com", "s3cret-passphrase")
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
		if _, err := svc.Verify(expiredToken); !errors.Is(err, domain.ErrUnauthorized) {
			t.Fatalf("expected ErrUnauthorized for expired token, got %v", err)
		}
	})
}
