package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareRejectsMissingHeader(t *testing.T) {
	handler := Middleware("secret", func(ctx context.Context, jti string) (bool, error) { return false, nil })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (sin header Authorization)", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	handler := Middleware("secret", func(ctx context.Context, jti string) (bool, error) { return false, nil })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer esto-no-es-un-jwt")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (token inválido)", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareRejectsRevokedToken(t *testing.T) {
	token, _, err := GenerateToken("secret", "user-123", "isabel@email.com", 24)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	handler := Middleware("secret", func(ctx context.Context, jti string) (bool, error) { return true, nil })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (token revocado)", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareAcceptsValidTokenAndSetsContext(t *testing.T) {
	token, _, err := GenerateToken("secret", "user-123", "isabel@email.com", 24)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	var gotUserID, gotEmail string
	handler := Middleware("secret", func(ctx context.Context, jti string) (bool, error) { return false, nil })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUserID = UserIDFromContext(r.Context())
			gotEmail = EmailFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (token válido)", rec.Code, http.StatusOK)
	}
	if gotUserID != "user-123" {
		t.Errorf("UserIDFromContext = %q, want %q", gotUserID, "user-123")
	}
	if gotEmail != "isabel@email.com" {
		t.Errorf("EmailFromContext = %q, want %q", gotEmail, "isabel@email.com")
	}
}
