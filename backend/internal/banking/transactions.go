package banking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/models"
)

// recordTransaction inserta el registro de auditoría en PostgreSQL. status puede
// ser "completed" o "failed"; tbTransferID es nil cuando la operación fue
// rechazada por TigerBeetle (nunca se creó una transferencia real).
func (s *Service) recordTransaction(ctx context.Context, tbTransferID *string, from, to, amount, txType, description, status string, userID *string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO transactions
			(id, tigerbeetle_transfer_id, from_account_number, to_account_number, amount, type, description, status, user_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		uuid.NewString(), tbTransferID, from, to, amount, txType, description, status, userID, time.Now(),
	)
	return err
}

// Deposit agrega fondos a una cuenta del usuario autenticado.
func (s *Service) Deposit(ctx context.Context, userID, accountNumber, amount, description string) (models.Account, error) {
	cents, err := ledger.DecimalStringToCents(amount)
	if err != nil {
		return models.Account{}, fmt.Errorf("invalid amount: %w", err)
	}

	row, err := s.getAccountRowByNumber(ctx, accountNumber)
	if err != nil {
		return models.Account{}, err
	}
	if row.UserID != userID {
		return models.Account{}, ErrForbidden
	}

	tbID, err := tbIDFromString(row.TBAccountID)
	if err != nil {
		return models.Account{}, err
	}

	transferID, err := s.ledger.Deposit(tbID, cents)
	if err != nil {
		_ = s.recordTransaction(ctx, nil, "EXTERNAL", accountNumber, ledger.CentsToDecimalString(cents), string(models.TransactionTypeDeposit), description, string(models.TransactionStatusFailed), &userID)
		return models.Account{}, fmt.Errorf("deposit failed: %w", err)
	}

	if err := s.recordTransaction(ctx, &transferID, "EXTERNAL", accountNumber, ledger.CentsToDecimalString(cents), string(models.TransactionTypeDeposit), description, string(models.TransactionStatusCompleted), &userID); err != nil {
		return models.Account{}, fmt.Errorf("recording transaction: %w", err)
	}

	return s.GetAccountByNumber(ctx, userID, accountNumber)
}

// Withdraw retira fondos de una cuenta del usuario autenticado. TigerBeetle
// rechaza la operación si no hay saldo suficiente (ledger.ErrInsufficientFunds).
func (s *Service) Withdraw(ctx context.Context, userID, accountNumber, amount, description string) (models.Account, error) {
	cents, err := ledger.DecimalStringToCents(amount)
	if err != nil {
		return models.Account{}, fmt.Errorf("invalid amount: %w", err)
	}

	row, err := s.getAccountRowByNumber(ctx, accountNumber)
	if err != nil {
		return models.Account{}, err
	}
	if row.UserID != userID {
		return models.Account{}, ErrForbidden
	}

	tbID, err := tbIDFromString(row.TBAccountID)
	if err != nil {
		return models.Account{}, err
	}

	transferID, err := s.ledger.Withdraw(tbID, cents)
	if err != nil {
		_ = s.recordTransaction(ctx, nil, accountNumber, "EXTERNAL", ledger.CentsToDecimalString(cents), string(models.TransactionTypeWithdrawal), description, string(models.TransactionStatusFailed), &userID)
		if errors.Is(err, ledger.ErrInsufficientFunds) {
			return models.Account{}, ErrInsufficientFunds
		}
		return models.Account{}, fmt.Errorf("withdrawal failed: %w", err)
	}

	if err := s.recordTransaction(ctx, &transferID, accountNumber, "EXTERNAL", ledger.CentsToDecimalString(cents), string(models.TransactionTypeWithdrawal), description, string(models.TransactionStatusCompleted), &userID); err != nil {
		return models.Account{}, fmt.Errorf("recording transaction: %w", err)
	}

	return s.GetAccountByNumber(ctx, userID, accountNumber)
}

// Transfer mueve fondos entre dos cuentas. La cuenta origen debe pertenecer al
// usuario autenticado; la cuenta destino solo debe existir.
func (s *Service) Transfer(ctx context.Context, userID, fromAccountNumber, toAccountNumber, amount, description string) (models.Account, error) {
	if fromAccountNumber == toAccountNumber {
		return models.Account{}, ErrSameAccount
	}

	cents, err := ledger.DecimalStringToCents(amount)
	if err != nil {
		return models.Account{}, fmt.Errorf("invalid amount: %w", err)
	}

	fromRow, err := s.getAccountRowByNumber(ctx, fromAccountNumber)
	if err != nil {
		return models.Account{}, err
	}
	if fromRow.UserID != userID {
		return models.Account{}, ErrForbidden
	}

	toRow, err := s.getAccountRowByNumber(ctx, toAccountNumber)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return models.Account{}, ErrDestinationNotFound
		}
		return models.Account{}, err
	}

	fromTBID, err := tbIDFromString(fromRow.TBAccountID)
	if err != nil {
		return models.Account{}, err
	}
	toTBID, err := tbIDFromString(toRow.TBAccountID)
	if err != nil {
		return models.Account{}, err
	}

	txType := string(models.TransactionTypeTransfer)
	if fromRow.UserID == toRow.UserID {
		txType = string(models.TransactionTypeInternalTransfer)
	}

	transferID, err := s.ledger.Transfer(fromTBID, toTBID, cents)
	if err != nil {
		_ = s.recordTransaction(ctx, nil, fromAccountNumber, toAccountNumber, ledger.CentsToDecimalString(cents), txType, description, string(models.TransactionStatusFailed), &userID)
		if errors.Is(err, ledger.ErrInsufficientFunds) {
			return models.Account{}, ErrInsufficientFunds
		}
		return models.Account{}, fmt.Errorf("transfer failed: %w", err)
	}

	if err := s.recordTransaction(ctx, &transferID, fromAccountNumber, toAccountNumber, ledger.CentsToDecimalString(cents), txType, description, string(models.TransactionStatusCompleted), &userID); err != nil {
		return models.Account{}, fmt.Errorf("recording transaction: %w", err)
	}

	return s.GetAccountByNumber(ctx, userID, fromAccountNumber)
}
