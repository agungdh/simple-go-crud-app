package service_test

import (
	"context"
	"errors"
	"testing"

	"simple-go-crud-app/internal/apperr"
	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/model"
	"simple-go-crud-app/internal/repository"
	"simple-go-crud-app/internal/service"
	"simple-go-crud-app/internal/testutil"
)

func TestEnsureAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("requires password when empty", func(t *testing.T) {
		users := repository.NewUserRepository(testutil.OpenDB(t))

		if _, err := service.EnsureAdmin(ctx, users, service.AdminSeed{
			Username: "admin",
			Name:     "Administrator",
		}); err == nil {
			t.Fatal("EnsureAdmin() error = nil, want missing password error")
		}
	})

	t.Run("rejects short password", func(t *testing.T) {
		users := repository.NewUserRepository(testutil.OpenDB(t))

		if _, err := service.EnsureAdmin(ctx, users, service.AdminSeed{
			Username: "admin",
			Password: "pendek",
			Name:     "Administrator",
		}); err == nil {
			t.Fatal("EnsureAdmin() error = nil, want short password error")
		}
	})

	t.Run("seeds once", func(t *testing.T) {
		users := repository.NewUserRepository(testutil.OpenDB(t))

		seeded, err := service.EnsureAdmin(ctx, users, service.AdminSeed{
			Username: "admin",
			Password: "rahasia123",
			Name:     "Administrator",
		})
		if err != nil {
			t.Fatalf("EnsureAdmin() error = %v", err)
		}
		if !seeded {
			t.Fatal("EnsureAdmin() seeded = false, want true")
		}

		user, err := users.GetByUsername(ctx, "admin")
		if err != nil {
			t.Fatalf("GetByUsername() error = %v", err)
		}
		if user.Role != model.RoleAdmin {
			t.Errorf("role = %q, want %q", user.Role, model.RoleAdmin)
		}
		if user.Password == "rahasia123" {
			t.Error("password was stored in plain text")
		}
		if !auth.VerifyPassword(user.Password, "rahasia123") {
			t.Error("stored hash does not verify the seeded password")
		}

		seeded, err = service.EnsureAdmin(ctx, users, service.AdminSeed{
			Username: "admin-lain",
			Password: "rahasia123",
			Name:     "Admin Lain",
		})
		if err != nil {
			t.Fatalf("second EnsureAdmin() error = %v", err)
		}
		if seeded {
			t.Fatal("second EnsureAdmin() seeded = true, want false")
		}

		if _, err := users.GetByUsername(ctx, "admin-lain"); !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("GetByUsername(admin-lain) error = %v, want ErrNotFound", err)
		}
	})
}
