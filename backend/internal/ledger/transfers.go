package ledger

import (
	"fmt"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// createSingleTransfer ejecuta una única transferencia y traduce el status a
// un error de dominio cuando corresponde. Devuelve el ID (como string decimal)
// de la transferencia creada.
func (c *Client) createSingleTransfer(t tb.Transfer) (string, error) {
	results, err := c.tb.CreateTransfers([]tb.Transfer{t})
	if err != nil {
		return "", fmt.Errorf("creating transfer: %w", err)
	}
	if len(results) == 0 {
		return t.ID.BigInt().String(), nil
	}
	switch results[0].Status {
	case tb.TransferCreated:
		return t.ID.BigInt().String(), nil
	case tb.TransferExceedsCredits:
		return "", ErrInsufficientFunds
	case tb.TransferDebitAccountNotFound, tb.TransferCreditAccountNotFound:
		return "", ErrAccountNotFound
	default:
		return "", fmt.Errorf("transfer rejected: %s", results[0].Status)
	}
}

// Deposit mueve fondos de la cuenta de control EXTERNAL hacia la cuenta del cliente.
func (c *Client) Deposit(customerAccountID tb.Uint128, amountCents int64) (string, error) {
	return c.createSingleTransfer(tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  ExternalAccountID,
		CreditAccountID: customerAccountID,
		Amount:          tb.ToUint128(uint64(amountCents)),
		Ledger:          LedgerUSD,
		Code:            CodeDeposit,
	})
}

// Withdraw mueve fondos de la cuenta del cliente hacia la cuenta de control EXTERNAL.
// TigerBeetle rechaza automáticamente la operación (TransferExceedsCredits) si la
// cuenta no tiene saldo suficiente, gracias al flag DebitsMustNotExceedCredits.
func (c *Client) Withdraw(customerAccountID tb.Uint128, amountCents int64) (string, error) {
	return c.createSingleTransfer(tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  customerAccountID,
		CreditAccountID: ExternalAccountID,
		Amount:          tb.ToUint128(uint64(amountCents)),
		Ledger:          LedgerUSD,
		Code:            CodeWithdrawal,
	})
}

// Transfer mueve fondos entre dos cuentas de cliente.
func (c *Client) Transfer(fromAccountID, toAccountID tb.Uint128, amountCents int64) (string, error) {
	return c.createSingleTransfer(tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  fromAccountID,
		CreditAccountID: toAccountID,
		Amount:          tb.ToUint128(uint64(amountCents)),
		Ledger:          LedgerUSD,
		Code:            CodeTransfer,
	})
}

// OpeningBalanceTransfer describe una transferencia de apertura a crear en batch durante el seeding.
type OpeningBalanceTransfer struct {
	AccountID   tb.Uint128
	AmountCents int64
}

// CreateOpeningBalances registra en batch el saldo inicial de cada cuenta sembrada,
// como transferencias EXTERNAL -> cuenta con Code=CodeOpeningBalance.
func (c *Client) CreateOpeningBalances(entries []OpeningBalanceTransfer) error {
	if len(entries) == 0 {
		return nil
	}
	transfers := make([]tb.Transfer, 0, len(entries))
	for _, e := range entries {
		if e.AmountCents <= 0 {
			continue
		}
		transfers = append(transfers, tb.Transfer{
			ID:              tb.ID(),
			DebitAccountID:  ExternalAccountID,
			CreditAccountID: e.AccountID,
			Amount:          tb.ToUint128(uint64(e.AmountCents)),
			Ledger:          LedgerUSD,
			Code:            CodeOpeningBalance,
		})
	}

	const chunkSize = 100
	for start := 0; start < len(transfers); start += chunkSize {
		end := start + chunkSize
		if end > len(transfers) {
			end = len(transfers)
		}
		results, err := c.tb.CreateTransfers(transfers[start:end])
		if err != nil {
			return fmt.Errorf("creating opening balance transfers: %w", err)
		}
		for _, r := range results {
			if r.Status != tb.TransferCreated && r.Status != tb.TransferExists {
				return fmt.Errorf("unexpected status creating opening balance: %s", r.Status)
			}
		}
	}
	return nil
}

// AccountHistoryEntry es una entrada simplificada del historial de transferencias
// de una cuenta, tal como la devuelve TigerBeetle.
type AccountHistoryEntry struct {
	TransferID string
	AmountCents int64
	Direction   string // "debit" o "credit" desde el punto de vista de la cuenta consultada
	Timestamp   uint64
}

// GetAccountHistory devuelve las transferencias más recientes de una cuenta (vía TigerBeetle).
func (c *Client) GetAccountHistory(accountID tb.Uint128, limit uint32) ([]AccountHistoryEntry, error) {
	filter := tb.AccountFilter{
		AccountID: accountID,
		Limit:     limit,
		Flags: tb.AccountFilterFlags{
			Debits:   true,
			Credits:  true,
			Reversed: true,
		}.ToUint32(),
	}
	transfers, err := c.tb.GetAccountTransfers(filter)
	if err != nil {
		return nil, fmt.Errorf("getting account transfers: %w", err)
	}

	entries := make([]AccountHistoryEntry, 0, len(transfers))
	for _, t := range transfers {
		direction := "credit"
		if t.DebitAccountID == accountID {
			direction = "debit"
		}
		entries = append(entries, AccountHistoryEntry{
			TransferID:  t.ID.BigInt().String(),
			AmountCents: uint128ToInt64(t.Amount),
			Direction:   direction,
			Timestamp:   t.Timestamp,
		})
	}
	return entries, nil
}

// BalancePoint es un punto de la serie de saldo en el tiempo de una cuenta.
type BalancePoint struct {
	TimestampNs uint64
	BalanceCents int64
}

// GetAccountBalanceHistory devuelve la serie histórica de saldo de una cuenta
// (requiere que la cuenta se haya creado con el flag History, como hacen
// todas las cuentas de cliente). Se usa para graficar la evolución del saldo.
func (c *Client) GetAccountBalanceHistory(accountID tb.Uint128, limit uint32) ([]BalancePoint, error) {
	filter := tb.AccountFilter{
		AccountID: accountID,
		Limit:     limit,
		Flags: tb.AccountFilterFlags{
			Debits:   true,
			Credits:  true,
			Reversed: true,
		}.ToUint32(),
	}
	balances, err := c.tb.GetAccountBalances(filter)
	if err != nil {
		return nil, fmt.Errorf("getting account balance history: %w", err)
	}

	points := make([]BalancePoint, 0, len(balances))
	for _, b := range balances {
		points = append(points, BalancePoint{
			TimestampNs:  b.Timestamp,
			BalanceCents: uint128ToInt64(b.CreditsPosted) - uint128ToInt64(b.DebitsPosted),
		})
	}
	// GetAccountBalances se pidió en orden inverso (más reciente primero);
	// para graficar conviene devolverlo en orden cronológico.
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	return points, nil
}
