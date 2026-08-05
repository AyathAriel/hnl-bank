package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Datos inválidos. Verifica el correo, la contraseña (mínimo 8 caracteres) y el nombre.")
		return
	}

	user, token, err := s.banking.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, banking.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "Ese correo ya está registrado.")
			return
		}
		writeError(w, http.StatusInternalServerError, "No se pudo completar el registro.")
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{Token: token, User: user})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Ingresa un correo y una contraseña válidos.")
		return
	}

	result, err := s.banking.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, banking.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "Correo o contraseña inválidos.")
			return
		}
		writeError(w, http.StatusInternalServerError, "No se pudo iniciar sesión.")
		return
	}

	if result.RequiresTOTP {
		writeJSON(w, http.StatusOK, AuthResponse{RequiresTOTP: true, PendingToken: result.PendingToken})
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: result.Token, User: result.User})
}

// handleVerify2FA canjea un pending_token + código TOTP por una sesión real.
// Vive en el grupo público de /api/auth (rate-limitado) porque, por
// definición, el usuario todavía no tiene una sesión completa en este punto.
func (s *Server) handleVerify2FA(w http.ResponseWriter, r *http.Request) {
	var req Verify2FARequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Ingresa un código de 6 dígitos válido.")
		return
	}

	user, token, err := s.banking.VerifyPendingLogin(r.Context(), req.PendingToken, req.Code)
	if err != nil {
		if errors.Is(err, banking.ErrInvalidTOTPCode) {
			writeError(w, http.StatusUnauthorized, "Código incorrecto.")
			return
		}
		writeError(w, http.StatusUnauthorized, "Sesión de verificación inválida o expirada. Inicia sesión de nuevo.")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: token, User: user})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	jti := auth.JTIFromContext(r.Context())
	if jti == "" {
		writeError(w, http.StatusBadRequest, "Falta el identificador del token.")
		return
	}
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWTExpiryHours) * time.Hour)
	if err := s.revocation.Revoke(r.Context(), jti, expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo cerrar la sesión.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
