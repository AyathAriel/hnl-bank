package auth

import (
	"bytes"
	"fmt"
	"image/png"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
)

const totpIssuer = "HNL Bank"

// GenerateTOTPSecret crea un secreto TOTP nuevo para accountName (el email del
// usuario) y devuelve tanto el secreto en base32 (por si el usuario prefiere
// ingresarlo a mano) como la URI otpauth:// que codifica el código QR.
func GenerateTOTPSecret(accountName string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", fmt.Errorf("generating totp secret: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTPCode verifica que code sea válido para secret en este momento
// (con la tolerancia de reloj por defecto de ±1 período de 30s).
func ValidateTOTPCode(secret, code string) bool {
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return valid
}

// GenerateQRCodePNG renderiza la URI otpauth:// como una imagen QR en PNG,
// lista para mostrarse en el frontend como data URL.
func GenerateQRCodePNG(otpauthURL string) ([]byte, error) {
	qr, err := qrcode.New(otpauthURL, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("generating qr code: %w", err)
	}
	img := qr.Image(256)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding qr png: %w", err)
	}
	return buf.Bytes(), nil
}
