package banking

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/models"
)

var (
	ErrTOTPAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrTOTPNotSetUp       = errors.New("two-factor authentication has not been set up")
	ErrInvalidTOTPCode    = errors.New("invalid verification code")
	ErrInvalidPendingToken = errors.New("invalid or expired pending token")
)

// Setup2FASecret genera un secreto TOTP nuevo para el usuario y lo guarda
// (sin activarlo todavía: hace falta confirmar con Enable2FA). Devuelve
// también la URI otpauth:// para renderizar el código QR.
func (s *Service) Setup2FASecret(ctx context.Context, userID string) (secret, otpauthURL string, err error) {
	var email string
	var alreadyEnabled bool
	if err := s.pool.QueryRow(ctx, `SELECT email, totp_enabled FROM users WHERE id = $1`, userID).Scan(&email, &alreadyEnabled); err != nil {
		return "", "", fmt.Errorf("querying user: %w", err)
	}
	if alreadyEnabled {
		return "", "", ErrTOTPAlreadyEnabled
	}

	secret, otpauthURL, err = auth.GenerateTOTPSecret(email)
	if err != nil {
		return "", "", fmt.Errorf("generating totp secret: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `UPDATE users SET totp_secret = $1, updated_at = now() WHERE id = $2`, secret, userID); err != nil {
		return "", "", fmt.Errorf("saving totp secret: %w", err)
	}

	return secret, otpauthURL, nil
}

// Enable2FA confirma la activación de 2FA: valida que code corresponda al
// secreto generado en Setup2FASecret y, si es así, activa totp_enabled.
func (s *Service) Enable2FA(ctx context.Context, userID, code string) error {
	var secret string
	var enabled bool
	err := s.pool.QueryRow(ctx, `SELECT totp_secret, totp_enabled FROM users WHERE id = $1`, userID).Scan(&secret, &enabled)
	if err != nil {
		return fmt.Errorf("querying user: %w", err)
	}
	if enabled {
		return ErrTOTPAlreadyEnabled
	}
	if secret == "" {
		return ErrTOTPNotSetUp
	}
	if !auth.ValidateTOTPCode(secret, code) {
		return ErrInvalidTOTPCode
	}

	if _, err := s.pool.Exec(ctx, `UPDATE users SET totp_enabled = true, updated_at = now() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("enabling totp: %w", err)
	}
	return nil
}

// Disable2FA desactiva 2FA, verificando la contraseña actual del usuario.
func (s *Service) Disable2FA(ctx context.Context, userID, password string) error {
	var hash string
	if err := s.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
		return fmt.Errorf("querying user: %w", err)
	}
	if !auth.CheckPassword(hash, password) {
		return ErrInvalidCredentials
	}

	if _, err := s.pool.Exec(ctx, `UPDATE users SET totp_enabled = false, totp_secret = NULL, updated_at = now() WHERE id = $1`, userID); err != nil {
		return fmt.Errorf("disabling totp: %w", err)
	}
	return nil
}

// TOTPStatus devuelve si el usuario tiene 2FA activado.
func (s *Service) TOTPStatus(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	if err := s.pool.QueryRow(ctx, `SELECT totp_enabled FROM users WHERE id = $1`, userID).Scan(&enabled); err != nil {
		return false, fmt.Errorf("querying user: %w", err)
	}
	return enabled, nil
}

// VerifyPendingLogin canjea un pending_token (emitido por Login cuando el
// usuario tiene 2FA) más un código TOTP válido por una sesión real.
func (s *Service) VerifyPendingLogin(ctx context.Context, pendingToken, code string) (models.User, string, error) {
	claims, err := auth.ParseToken(s.jwtSecret, pendingToken)
	if err != nil || claims.Purpose != auth.PurposePending2FA {
		return models.User{}, "", ErrInvalidPendingToken
	}

	var user models.User
	var secret string
	var enabled bool
	err = s.pool.QueryRow(ctx,
		`SELECT id, email, full_name, created_at, totp_secret, totp_enabled FROM users WHERE id = $1`,
		claims.UserID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.CreatedAt, &secret, &enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, "", ErrInvalidPendingToken
		}
		return models.User{}, "", fmt.Errorf("querying user: %w", err)
	}
	user.TOTPEnabled = enabled

	if !enabled || secret == "" || !auth.ValidateTOTPCode(secret, code) {
		return models.User{}, "", ErrInvalidTOTPCode
	}

	token, _, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Email, s.jwtExpiryHours)
	if err != nil {
		return models.User{}, "", fmt.Errorf("generating token: %w", err)
	}
	return user, token, nil
}
