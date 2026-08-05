package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
)

func (s *Server) handle2FAStatus(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	enabled, err := s.banking.TOTPStatus(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo consultar el estado de 2FA.")
		return
	}
	writeJSON(w, http.StatusOK, TOTPStatusResponse{Enabled: enabled})
}

func (s *Server) handle2FASetup(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	secret, otpauthURL, err := s.banking.Setup2FASecret(r.Context(), userID)
	if err != nil {
		if errors.Is(err, banking.ErrTOTPAlreadyEnabled) {
			writeError(w, http.StatusConflict, "Ya tienes 2FA activado.")
			return
		}
		writeError(w, http.StatusInternalServerError, "No se pudo iniciar la configuración de 2FA.")
		return
	}

	qrPNG, err := auth.GenerateQRCodePNG(otpauthURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudo generar el código QR.")
		return
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)

	writeJSON(w, http.StatusOK, Setup2FAResponse{Secret: secret, QRCodeDataURL: dataURL})
}

func (s *Server) handle2FAEnable(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req Enable2FARequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Ingresa un código de 6 dígitos válido.")
		return
	}

	if err := s.banking.Enable2FA(r.Context(), userID, req.Code); err != nil {
		switch {
		case errors.Is(err, banking.ErrTOTPAlreadyEnabled):
			writeError(w, http.StatusConflict, "Ya tienes 2FA activado.")
		case errors.Is(err, banking.ErrTOTPNotSetUp):
			writeError(w, http.StatusBadRequest, "Primero genera un código QR desde /api/auth/2fa/setup.")
		case errors.Is(err, banking.ErrInvalidTOTPCode):
			writeError(w, http.StatusBadRequest, "Código incorrecto. Verifica la hora de tu celular e intenta de nuevo.")
		default:
			writeError(w, http.StatusInternalServerError, "No se pudo activar 2FA.")
		}
		return
	}

	writeJSON(w, http.StatusOK, TOTPStatusResponse{Enabled: true})
}

func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req Disable2FARequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Ingresa tu contraseña actual.")
		return
	}

	if err := s.banking.Disable2FA(r.Context(), userID, req.Password); err != nil {
		if errors.Is(err, banking.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "Contraseña incorrecta.")
			return
		}
		writeError(w, http.StatusInternalServerError, "No se pudo desactivar 2FA.")
		return
	}

	writeJSON(w, http.StatusOK, TOTPStatusResponse{Enabled: false})
}
