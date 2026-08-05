package banking

import (
	"context"
	"fmt"

	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/models"
)

// userAccountNumbers devuelve todos los números de cuenta del usuario.
func (s *Service) userAccountNumbers(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT account_number FROM accounts WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("querying account numbers: %w", err)
	}
	defer rows.Close()

	var numbers []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		numbers = append(numbers, n)
	}
	return numbers, rows.Err()
}

// GetHistory devuelve el historial paginado de transacciones del usuario,
// opcionalmente filtrado a una sola cuenta.
func (s *Service) GetHistory(ctx context.Context, userID, accountNumberFilter string, page, pageSize int) ([]models.Transaction, models.Pagination, error) {
	numbers, err := s.userAccountNumbers(ctx, userID)
	if err != nil {
		return nil, models.Pagination{}, err
	}
	if len(numbers) == 0 {
		return []models.Transaction{}, models.Pagination{Page: page, PageSize: pageSize, Total: 0}, nil
	}

	if accountNumberFilter != "" {
		owned := false
		for _, n := range numbers {
			if n == accountNumberFilter {
				owned = true
				break
			}
		}
		if !owned {
			return nil, models.Pagination{}, ErrForbidden
		}
		numbers = []string{accountNumberFilter}
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM transactions WHERE from_account_number = ANY($1) OR to_account_number = ANY($1)`,
		numbers,
	).Scan(&total); err != nil {
		return nil, models.Pagination{}, fmt.Errorf("counting transactions: %w", err)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, from_account_number, to_account_number, amount::text, type, description, status, created_at
		 FROM transactions
		 WHERE from_account_number = ANY($1) OR to_account_number = ANY($1)
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		numbers, pageSize, offset,
	)
	if err != nil {
		return nil, models.Pagination{}, fmt.Errorf("querying transactions: %w", err)
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.FromAccountNumber, &t.ToAccountNumber, &t.Amount, &t.Type, &t.Description, &t.Status, &t.CreatedAt); err != nil {
			return nil, models.Pagination{}, fmt.Errorf("scanning transaction: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, models.Pagination{}, err
	}
	if txs == nil {
		txs = []models.Transaction{}
	}

	return txs, models.Pagination{Page: page, PageSize: pageSize, Total: total}, nil
}

// DashboardData agrupa la información resumida para la vista principal.
type DashboardData struct {
	Accounts            []models.Account   `json:"accounts"`
	RecentTransactions  []models.Transaction `json:"recent_transactions"`
	TotalBalance        string             `json:"total_balance"`
}

func (s *Service) GetDashboard(ctx context.Context, userID string) (DashboardData, error) {
	accounts, err := s.GetAccounts(ctx, userID)
	if err != nil {
		return DashboardData{}, err
	}

	var totalCents int64
	for _, a := range accounts {
		totalCents += a.BalanceCents
	}

	recent, _, err := s.GetHistory(ctx, userID, "", 1, 10)
	if err != nil {
		return DashboardData{}, err
	}

	return DashboardData{
		Accounts:           accounts,
		RecentTransactions: recent,
		TotalBalance:       ledger.CentsToDecimalString(totalCents),
	}, nil
}
