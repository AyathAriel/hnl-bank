package auth

import (
	"testing"
	"time"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret"
	token, jti, err := GenerateToken(secret, "user-123", "isabel@email.com", 24)
	if err != nil {
		t.Fatalf("GenerateToken devolvió error: %v", err)
	}
	if jti == "" {
		t.Fatal("GenerateToken debería devolver un jti no vacío")
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken devolvió error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}
	if claims.Email != "isabel@email.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "isabel@email.com")
	}
	if claims.ID != jti {
		t.Errorf("claims.ID = %q, want %q (el jti debe viajar en el token)", claims.ID, jti)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, _, err := GenerateToken("secret-a", "user-123", "isabel@email.com", 24)
	if err != nil {
		t.Fatalf("GenerateToken devolvió error: %v", err)
	}
	if _, err := ParseToken("secret-b", token); err == nil {
		t.Error("ParseToken debería rechazar un token firmado con otro secreto")
	}
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	// Genera un token que ya expiró (0 horas de vigencia, más un instante de margen).
	token, _, err := GenerateToken("test-secret", "user-123", "isabel@email.com", 0)
	if err != nil {
		t.Fatalf("GenerateToken devolvió error: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)

	if _, err := ParseToken("test-secret", token); err == nil {
		t.Error("ParseToken debería rechazar un token expirado")
	}
}

func TestParseTokenRejectsGarbage(t *testing.T) {
	if _, err := ParseToken("test-secret", "no-soy-un-jwt"); err == nil {
		t.Error("ParseToken debería rechazar una cadena que no es un JWT válido")
	}
}
