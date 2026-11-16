package router_test

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"simple-go-crud-app/internal/auth"
	"simple-go-crud-app/internal/bidang"
	"simple-go-crud-app/internal/shared/entity"
	"simple-go-crud-app/internal/shared/httpx"
)

func loginBody(username, password string) string {
	return fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
}

func refreshBody(token string) string {
	return fmt.Sprintf(`{"refresh_token":%q}`, token)
}

func TestAuthLogin(t *testing.T) {
	env := newTestEnv(t)

	t.Run("success", func(t *testing.T) {
		rec := env.doAnon(http.MethodPost, "/auth/login", loginBody(testAdminUsername, testPassword))
		requireStatus(t, rec, http.StatusOK)

		body := decodeJSON[auth.TokenResponse](t, rec)
		if body.AccessToken == "" {
			t.Error("access_token is empty")
		}
		if body.RefreshToken == "" {
			t.Error("refresh_token is empty")
		}
		if body.TokenType != "Bearer" {
			t.Errorf("token_type = %q, want Bearer", body.TokenType)
		}
		if body.ExpiresIn != int64(testAccessTTL.Seconds()) {
			t.Errorf("expires_in = %d, want %d", body.ExpiresIn, int64(testAccessTTL.Seconds()))
		}
		if body.RefreshExpiresIn != int64(testRefreshTTL.Seconds()) {
			t.Errorf("refresh_expires_in = %d, want %d", body.RefreshExpiresIn, int64(testRefreshTTL.Seconds()))
		}
		if body.User.Username != testAdminUsername || body.User.Role != auth.RoleAdmin {
			t.Errorf("user = %+v, want admin role and username %q", body.User, testAdminUsername)
		}
		if strings.Contains(rec.Body.String(), testPassword) {
			t.Error("response leaks the password")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		rec := env.doAnon(http.MethodPost, "/auth/login", loginBody(testAdminUsername, "salah-sekali"))
		requireStatus(t, rec, http.StatusUnauthorized)
		if decodeJSON[httpx.ErrorResponse](t, rec).Error == "" {
			t.Error("401 response has an empty error message")
		}
	})

	t.Run("unknown username uses the same message", func(t *testing.T) {
		wrongPassword := env.doAnon(http.MethodPost, "/auth/login", loginBody(testAdminUsername, "salah-sekali"))
		unknownUser := env.doAnon(http.MethodPost, "/auth/login", loginBody("tidak-ada", testPassword))

		requireStatus(t, wrongPassword, http.StatusUnauthorized)
		requireStatus(t, unknownUser, http.StatusUnauthorized)
		if wrongPassword.Body.String() != unknownUser.Body.String() {
			t.Errorf(
				"messages differ: wrong password %q, unknown user %q",
				wrongPassword.Body.String(), unknownUser.Body.String(),
			)
		}
	})

	t.Run("validation", func(t *testing.T) {
		rec := env.doAnon(http.MethodPost, "/auth/login", `{}`)
		requireValidationFields(t, rec, "username", "password")

		rec = env.doAnon(http.MethodPost, "/auth/login", `{"username":"   ","password":""}`)
		requireValidationFields(t, rec, "username", "password")

		rec = env.doAnon(http.MethodPost, "/auth/login", `{"username":1,"password":false}`)
		requireValidationFields(t, rec, "username", "password")

		rec = env.doAnon(http.MethodPost, "/auth/login", `{"username":"admin","password":"rahasia123","lain":"x"}`)
		requireValidationFields(t, rec, "lain")

		rec = env.doAnon(http.MethodPost, "/auth/login", `{"username":"`+strings.Repeat("a", 51)+`","password":"rahasia123"}`)
		requireValidationFields(t, rec, "username")

		rec = env.doAnon(http.MethodPost, "/auth/login", `{"username":"admin","password":"`+strings.Repeat("a", 73)+`"}`)
		requireValidationFields(t, rec, "password")
	})

	t.Run("bad body", func(t *testing.T) {
		rec := env.doAnon(http.MethodPost, "/auth/login", "{")
		requireStatus(t, rec, http.StatusBadRequest)
	})
}

func TestAuthRefresh(t *testing.T) {
	env := newTestEnv(t)
	first := env.login(testAdminUsername, testPassword)

	rec := env.doAnon(http.MethodPost, "/auth/refresh", refreshBody(first.RefreshToken))
	requireStatus(t, rec, http.StatusOK)

	second := decodeJSON[auth.TokenResponse](t, rec)
	if second.AccessToken == "" || second.RefreshToken == "" {
		t.Fatal("refresh response has an empty token")
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}
	if second.User.Username != testAdminUsername {
		t.Errorf("user = %+v, want username %q", second.User, testAdminUsername)
	}

	// The new access token works.
	rec = env.doAs(second.AccessToken, http.MethodGet, "/bidang", "", nil)
	requireStatus(t, rec, http.StatusOK)

	// The consumed refresh token cannot be replayed.
	rec = env.doAnon(http.MethodPost, "/auth/refresh", refreshBody(first.RefreshToken))
	requireStatus(t, rec, http.StatusUnauthorized)

	rec = env.doAnon(http.MethodPost, "/auth/refresh", refreshBody("not-a-real-token"))
	requireStatus(t, rec, http.StatusUnauthorized)

	rec = env.doAnon(http.MethodPost, "/auth/refresh", `{}`)
	requireValidationFields(t, rec, "refresh_token")

	rec = env.doAnon(http.MethodPost, "/auth/refresh", `{"refresh_token":123}`)
	requireValidationFields(t, rec, "refresh_token")
}

func TestAuthRefreshExpiredToken(t *testing.T) {
	env := newTestEnv(t)
	tokens := env.login(testAdminUsername, testPassword)

	hash := auth.HashRefreshToken(tokens.RefreshToken)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	if _, err := env.db.NewUpdate().
		Model((*auth.RefreshToken)(nil)).
		Set("expires_at = ?", past).
		Where("token_hash = ?", hash).
		Exec(context.Background()); err != nil {
		t.Fatalf("expire refresh token: %v", err)
	}

	rec := env.doAnon(http.MethodPost, "/auth/refresh", refreshBody(tokens.RefreshToken))
	requireStatus(t, rec, http.StatusUnauthorized)
}

func TestAuthLogout(t *testing.T) {
	env := newTestEnv(t)
	tokens := env.login(testAdminUsername, testPassword)

	rec := env.doAnon(http.MethodPost, "/auth/logout", refreshBody(tokens.RefreshToken))
	requireStatus(t, rec, http.StatusUnauthorized)

	rec = env.doAs(tokens.AccessToken, http.MethodPost, "/auth/logout", `{}`, nil)
	requireValidationFields(t, rec, "refresh_token")

	rec = env.doAs(tokens.AccessToken, http.MethodPost, "/auth/logout", refreshBody(tokens.RefreshToken), nil)
	requireStatus(t, rec, http.StatusNoContent)
	requireEmptyBody(t, rec)

	// The revoked refresh token can no longer be used.
	rec = env.doAnon(http.MethodPost, "/auth/refresh", refreshBody(tokens.RefreshToken))
	requireStatus(t, rec, http.StatusUnauthorized)

	// Logout is idempotent, and the access token stays valid until it expires.
	rec = env.doAs(tokens.AccessToken, http.MethodPost, "/auth/logout", refreshBody(tokens.RefreshToken), nil)
	requireStatus(t, rec, http.StatusNoContent)

	rec = env.doAs(tokens.AccessToken, http.MethodGet, "/bidang", "", nil)
	requireStatus(t, rec, http.StatusOK)
}

func TestAuthAccessControl(t *testing.T) {
	env := newTestEnv(t)

	t.Run("missing or invalid token", func(t *testing.T) {
		rec := env.doAnon(http.MethodGet, "/bidang", "")
		requireStatus(t, rec, http.StatusUnauthorized)

		rec = env.doAs("not-a-token", http.MethodGet, "/bidang", "", nil)
		requireStatus(t, rec, http.StatusUnauthorized)

		rec = env.doAs("", http.MethodGet, "/bidang", "", map[string]string{
			"Authorization": "Basic abc",
		})
		requireStatus(t, rec, http.StatusUnauthorized)

		rec = env.doAs("", http.MethodGet, "/bidang", "", map[string]string{
			"Authorization": "Bearer",
		})
		requireStatus(t, rec, http.StatusUnauthorized)
	})

	t.Run("user can read", func(t *testing.T) {
		for _, path := range []string{"/bidang", "/pegawai"} {
			rec := env.doAs(env.userToken, http.MethodGet, path, "", nil)
			requireStatus(t, rec, http.StatusOK)
		}
	})

	t.Run("user cannot write", func(t *testing.T) {
		tests := []struct {
			method string
			path   string
			body   string
		}{
			{http.MethodPost, "/bidang", `{"bidang":"Keuangan"}`},
			{http.MethodPut, "/bidang/1", `{"bidang":"Keuangan"}`},
			{http.MethodDelete, "/bidang/1", ""},
			{http.MethodPost, "/pegawai", `{}`},
			{http.MethodPut, "/pegawai/1", `{}`},
			{http.MethodDelete, "/pegawai/1", ""},
		}
		for _, test := range tests {
			rec := env.doAs(env.userToken, test.method, test.path, test.body, nil)
			requireStatus(t, rec, http.StatusForbidden)
		}
	})

	t.Run("admin can write", func(t *testing.T) {
		rec := env.do(http.MethodPost, "/bidang", `{"bidang":"Keuangan"}`)
		requireStatus(t, rec, http.StatusCreated)

		rec = env.do(http.MethodGet, "/bidang", "")
		requireStatus(t, rec, http.StatusOK)
		list := decodeJSON[httpx.ListEnvelope[bidang.Response]](t, rec)
		if len(list.Data) != 1 {
			t.Fatalf("bidang count = %d, want 1", len(list.Data))
		}
		id := list.Data[0].ID

		rec = env.do(http.MethodPut, fmt.Sprintf("/bidang/%d", id), `{"bidang":"Keuangan Pusat"}`)
		requireStatus(t, rec, http.StatusOK)

		rec = env.do(http.MethodDelete, fmt.Sprintf("/bidang/%d", id), "")
		requireStatus(t, rec, http.StatusNoContent)
	})
}

func TestAuthExpiredAccessToken(t *testing.T) {
	env := newTestEnv(t)

	users := auth.NewUserRepository(env.db)
	admin, err := users.GetByID(context.Background(), env.adminID)
	if err != nil {
		t.Fatalf("load admin: %v", err)
	}

	now := time.Now().UTC()
	claims := auth.Claims{
		Username: admin.Username,
		Role:     admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    auth.Issuer,
			Subject:   strconv.FormatInt(admin.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	rec := env.doAs(signed, http.MethodGet, "/bidang", "", nil)
	requireStatus(t, rec, http.StatusUnauthorized)
}

func TestAuthSoftDeletedUser(t *testing.T) {
	env := newTestEnv(t)
	tokens := env.login(testAdminUsername, testPassword)

	if _, err := env.db.NewUpdate().
		Model((*auth.User)(nil)).
		Set("deleted_at = ?", entity.NowMillis()).
		Where("id = ?", env.adminID).
		Exec(context.Background()); err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	rec := env.doAs(tokens.AccessToken, http.MethodGet, "/bidang", "", nil)
	requireStatus(t, rec, http.StatusUnauthorized)
}
