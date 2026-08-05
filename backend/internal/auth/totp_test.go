package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestGenerateTOTPSecret(t *testing.T) {
	secret, otpauthURL, err := GenerateTOTPSecret("isabel@email.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret devolvió error: %v", err)
	}
	if secret == "" {
		t.Error("el secreto no debería estar vacío")
	}
	if !strings.HasPrefix(otpauthURL, "otpauth://totp/") {
		t.Errorf("otpauthURL = %q, debería empezar con otpauth://totp/", otpauthURL)
	}
	if !strings.Contains(otpauthURL, "HNL") {
		t.Error("otpauthURL debería incluir el issuer (HNL Bank)")
	}
}

func TestValidateTOTPCodeAcceptsCurrentCode(t *testing.T) {
	secret, _, err := GenerateTOTPSecret("isabel@email.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generando código de prueba: %v", err)
	}

	if !ValidateTOTPCode(secret, code) {
		t.Error("ValidateTOTPCode debería aceptar un código recién generado para ese secreto")
	}
}

func TestValidateTOTPCodeRejectsWrongCode(t *testing.T) {
	secret, _, err := GenerateTOTPSecret("isabel@email.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}

	if ValidateTOTPCode(secret, "000000") {
		t.Error("ValidateTOTPCode no debería aceptar un código arbitrario/incorrecto")
	}
}

func TestGenerateQRCodePNGProducesValidPNG(t *testing.T) {
	_, otpauthURL, err := GenerateTOTPSecret("isabel@email.com")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}

	png, err := GenerateQRCodePNG(otpauthURL)
	if err != nil {
		t.Fatalf("GenerateQRCodePNG devolvió error: %v", err)
	}
	if len(png) < 8 {
		t.Fatal("el PNG generado es demasiado pequeño para ser válido")
	}
	// firma estándar de PNG: 89 50 4E 47 0D 0A 1A 0A
	sig := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i, b := range sig {
		if png[i] != b {
			t.Fatalf("el archivo generado no tiene la firma PNG esperada en el byte %d", i)
		}
	}
}
