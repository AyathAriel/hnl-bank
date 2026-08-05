package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
	"github.com/hnl/bank-backend/internal/ledger"
)

// notificationEvent es el payload que reciben los clientes conectados por
// WebSocket cuando se completa una transacción sobre una de sus cuentas.
type notificationEvent struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Account interface{} `json:"account"`
}

func (s *Server) notify(userID, eventType, message string, account interface{}) {
	if s.hub == nil {
		return
	}
	s.hub.Notify(userID, notificationEvent{Type: eventType, Message: message, Account: account})
}

// mapBankingError traduce errores de dominio a códigos HTTP + mensaje.
func mapBankingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, banking.ErrAccountNotFound), errors.Is(err, banking.ErrDestinationNotFound):
		writeError(w, http.StatusNotFound, "No se encontró la cuenta indicada.")
	case errors.Is(err, banking.ErrForbidden):
		writeError(w, http.StatusForbidden, "Esa cuenta no te pertenece.")
	case errors.Is(err, banking.ErrSameAccount):
		writeError(w, http.StatusBadRequest, "No puedes transferir a la misma cuenta.")
	case errors.Is(err, banking.ErrInsufficientFunds):
		writeError(w, http.StatusUnprocessableEntity, "Fondos insuficientes para completar la operación.")
	default:
		writeError(w, http.StatusBadRequest, "No se pudo completar la operación. Verifica los datos e intenta de nuevo.")
	}
}

func (s *Server) handleDeposit(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req DepositRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Datos inválidos. Revisa el formulario e intenta de nuevo.")
		return
	}
	if !ledger.AmountPattern.MatchString(req.Amount) {
		writeError(w, http.StatusBadRequest, "El monto debe ser un número positivo con hasta 2 decimales.")
		return
	}

	account, err := s.banking.Deposit(r.Context(), userID, req.AccountNumber, req.Amount, req.Description)
	if err != nil {
		mapBankingError(w, err)
		return
	}
	s.notify(userID, "deposit", "Depósito de $"+req.Amount+" acreditado en "+req.AccountNumber, account)
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req WithdrawRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Datos inválidos. Revisa el formulario e intenta de nuevo.")
		return
	}
	if !ledger.AmountPattern.MatchString(req.Amount) {
		writeError(w, http.StatusBadRequest, "El monto debe ser un número positivo con hasta 2 decimales.")
		return
	}

	account, err := s.banking.Withdraw(r.Context(), userID, req.AccountNumber, req.Amount, req.Description)
	if err != nil {
		mapBankingError(w, err)
		return
	}
	s.notify(userID, "withdrawal", "Retiro de $"+req.Amount+" desde "+req.AccountNumber, account)
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req TransferRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Datos inválidos. Revisa el formulario e intenta de nuevo.")
		return
	}
	if !ledger.AmountPattern.MatchString(req.Amount) {
		writeError(w, http.StatusBadRequest, "El monto debe ser un número positivo con hasta 2 decimales.")
		return
	}

	account, err := s.banking.Transfer(r.Context(), userID, req.FromAccountNumber, req.ToAccountNumber, req.Amount, req.Description)
	if err != nil {
		mapBankingError(w, err)
		return
	}
	s.notify(userID, "transfer", "Transferiste $"+req.Amount+" a la cuenta "+req.ToAccountNumber, account)
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	accountNumber := r.URL.Query().Get("account")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	txs, pagination, err := s.banking.GetHistory(r.Context(), userID, accountNumber, page, pageSize)
	if err != nil {
		mapBankingError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": txs,
		"pagination":   pagination,
	})
}
