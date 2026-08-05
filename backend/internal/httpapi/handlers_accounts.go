package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
)

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	accounts, err := s.banking.GetAccounts(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "No se pudieron obtener las cuentas.")
		return
	}
	writeJSON(w, http.StatusOK, accounts)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	number := chi.URLParam(r, "number")

	account, err := s.banking.GetAccountByNumber(r.Context(), userID, number)
	if err != nil {
		switch {
		case errors.Is(err, banking.ErrAccountNotFound):
			writeError(w, http.StatusNotFound, "No se encontró la cuenta.")
		case errors.Is(err, banking.ErrForbidden):
			writeError(w, http.StatusForbidden, "Esa cuenta no te pertenece.")
		default:
			writeError(w, http.StatusInternalServerError, "No se pudo obtener la cuenta.")
		}
		return
	}
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleBalanceHistory(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	number := chi.URLParam(r, "number")

	points, err := s.banking.GetBalanceHistory(r.Context(), userID, number)
	if err != nil {
		switch {
		case errors.Is(err, banking.ErrAccountNotFound):
			writeError(w, http.StatusNotFound, "No se encontró la cuenta.")
		case errors.Is(err, banking.ErrForbidden):
			writeError(w, http.StatusForbidden, "Esa cuenta no te pertenece.")
		default:
			writeError(w, http.StatusInternalServerError, "No se pudo obtener el historial de saldo.")
		}
		return
	}
	writeJSON(w, http.StatusOK, points)
}
