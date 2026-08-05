package banking

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	tb "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/models"
)

type accountRow struct {
	ID            string
	AccountNumber string
	UserID        string
	TBAccountID   string
	Currency      string
	AccountType   string
	CreatedAt     time.Time
}

func tbIDFromString(s string) (tb.Uint128, error) {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return tb.Uint128{}, fmt.Errorf("invalid tigerbeetle id: %q", s)
	}
	return tb.BigIntToUint128(n), nil
}

// GetAccounts devuelve todas las cuentas del usuario con su saldo actual (desde TigerBeetle).
func (s *Service) GetAccounts(ctx context.Context, userID string) ([]models.Account, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, account_number, user_id, tigerbeetle_account_id::text, currency, account_type, created_at
		 FROM accounts WHERE user_id = $1 ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying accounts: %w", err)
	}
	defer rows.Close()

	accounts := []models.Account{}
	var tbIDs []tb.Uint128
	for rows.Next() {
		var a models.Account
		var tbIDStr string
		if err := rows.Scan(&a.ID, &a.AccountNumber, &a.UserID, &tbIDStr, &a.Currency, &a.AccountType, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning account: %w", err)
		}
		a.TigerBeetleID = tbIDStr
		tbID, err := tbIDFromString(tbIDStr)
		if err != nil {
			return nil, err
		}
		tbIDs = append(tbIDs, tbID)
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	balances, err := s.ledger.GetBalancesCents(tbIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching balances: %w", err)
	}
	for i := range accounts {
		tbID, _ := tbIDFromString(accounts[i].TigerBeetleID)
		cents := balances[tbID]
		accounts[i].BalanceCents = cents
		accounts[i].Balance = ledger.CentsToDecimalString(cents)
	}

	return accounts, nil
}

// getAccountRowByNumber busca una cuenta por su número, sin verificar propietario.
func (s *Service) getAccountRowByNumber(ctx context.Context, accountNumber string) (accountRow, error) {
	var a accountRow
	err := s.pool.QueryRow(ctx,
		`SELECT id, account_number, user_id, tigerbeetle_account_id::text, currency, account_type, created_at
		 FROM accounts WHERE account_number = $1`,
		accountNumber,
	).Scan(&a.ID, &a.AccountNumber, &a.UserID, &a.TBAccountID, &a.Currency, &a.AccountType, &a.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return accountRow{}, ErrAccountNotFound
		}
		return accountRow{}, fmt.Errorf("querying account: %w", err)
	}
	return a, nil
}

// GetAccountByNumber devuelve el detalle + saldo de una cuenta, verificando que
// pertenezca al usuario autenticado.
func (s *Service) GetAccountByNumber(ctx context.Context, userID, accountNumber string) (models.Account, error) {
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
	cents, err := s.ledger.GetBalanceCents(tbID)
	if err != nil {
		return models.Account{}, fmt.Errorf("fetching balance: %w", err)
	}

	return models.Account{
		ID:            row.ID,
		AccountNumber: row.AccountNumber,
		UserID:        row.UserID,
		Currency:      row.Currency,
		AccountType:   models.AccountType(row.AccountType),
		CreatedAt:     row.CreatedAt,
		BalanceCents:  cents,
		Balance:       ledger.CentsToDecimalString(cents),
	}, nil
}

// BalancePoint es un punto de la serie histórica de saldo de una cuenta,
// listo para graficar (timestamp ya convertido a time.Time y monto ya
// formateado como decimal).
type BalancePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Balance   string    `json:"balance"`
}

// GetBalanceHistory devuelve la evolución del saldo de una cuenta propia del usuario.
func (s *Service) GetBalanceHistory(ctx context.Context, userID, accountNumber string) ([]BalancePoint, error) {
	row, err := s.getAccountRowByNumber(ctx, accountNumber)
	if err != nil {
		return nil, err
	}
	if row.UserID != userID {
		return nil, ErrForbidden
	}

	tbID, err := tbIDFromString(row.TBAccountID)
	if err != nil {
		return nil, err
	}

	points, err := s.ledger.GetAccountBalanceHistory(tbID, 100)
	if err != nil {
		return nil, fmt.Errorf("fetching balance history: %w", err)
	}

	result := make([]BalancePoint, 0, len(points))
	for _, p := range points {
		result = append(result, BalancePoint{
			// Los timestamps de TigerBeetle son nanosegundos desde epoch.
			Timestamp: time.Unix(0, int64(p.TimestampNs)),
			Balance:   ledger.CentsToDecimalString(p.BalanceCents),
		})
	}
	return result, nil
}
