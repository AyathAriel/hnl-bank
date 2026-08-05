package ledger

import (
	"fmt"

	tb "github.com/tigerbeetle/tigerbeetle-go"
)

// uint128ToInt64 convierte un Uint128 de TigerBeetle a int64. Es seguro para
// montos bancarios reales (muy por debajo de 2^63 centavos).
func uint128ToInt64(v tb.Uint128) int64 {
	return v.BigInt().Int64()
}

// EnsureExternalAccount crea la cuenta de control EXTERNAL si todavía no existe.
// Es idempotente: si ya existe, TigerBeetle devuelve AccountExists y se ignora.
func (c *Client) EnsureExternalAccount() error {
	results, err := c.tb.CreateAccounts([]tb.Account{
		{
			ID:     ExternalAccountID,
			Ledger: LedgerUSD,
			Code:   CodeExternalAccount,
			Flags:  tb.AccountFlags{History: true}.ToUint16(),
		},
	})
	if err != nil {
		return fmt.Errorf("creating external account: %w", err)
	}
	for _, r := range results {
		if r.Status != tb.AccountCreated && r.Status != tb.AccountExists {
			return fmt.Errorf("unexpected status creating external account: %s", r.Status)
		}
	}
	return nil
}

// NewCustomerAccountID genera un ID nuevo, único y ordenable en el tiempo para
// una cuenta de cliente.
func NewCustomerAccountID() tb.Uint128 {
	return tb.ID()
}

// CreateCustomerAccounts crea en batch una o más cuentas de cliente. Cada
// cuenta lleva DebitsMustNotExceedCredits=true, lo que hace que el propio
// ledger rechace sobregiros (defensa adicional a la validación de la app), y
// History=true para poder graficar el saldo a lo largo del tiempo.
func (c *Client) CreateCustomerAccounts(accounts []struct {
	ID          tb.Uint128
	AccountType string
}) error {
	if len(accounts) == 0 {
		return nil
	}
	batch := make([]tb.Account, 0, len(accounts))
	for _, a := range accounts {
		batch = append(batch, tb.Account{
			ID:     a.ID,
			Ledger: LedgerUSD,
			Code:   AccountTypeToCode(a.AccountType),
			Flags: tb.AccountFlags{
				DebitsMustNotExceedCredits: true,
				History:                    true,
			}.ToUint16(),
		})
	}

	const chunkSize = 100
	for start := 0; start < len(batch); start += chunkSize {
		end := start + chunkSize
		if end > len(batch) {
			end = len(batch)
		}
		results, err := c.tb.CreateAccounts(batch[start:end])
		if err != nil {
			return fmt.Errorf("creating customer accounts: %w", err)
		}
		for _, r := range results {
			if r.Status != tb.AccountCreated && r.Status != tb.AccountExists {
				return fmt.Errorf("unexpected status creating customer account: %s", r.Status)
			}
		}
	}
	return nil
}

// GetBalanceCents obtiene el saldo actual (creditsPosted - debitsPosted) de una cuenta.
func (c *Client) GetBalanceCents(accountID tb.Uint128) (int64, error) {
	balances, err := c.GetBalancesCents([]tb.Uint128{accountID})
	if err != nil {
		return 0, err
	}
	balance, ok := balances[accountID]
	if !ok {
		return 0, ErrAccountNotFound
	}
	return balance, nil
}

// GetBalancesCents obtiene el saldo actual de varias cuentas en una sola llamada.
func (c *Client) GetBalancesCents(accountIDs []tb.Uint128) (map[tb.Uint128]int64, error) {
	if len(accountIDs) == 0 {
		return map[tb.Uint128]int64{}, nil
	}
	accounts, err := c.tb.LookupAccounts(accountIDs)
	if err != nil {
		return nil, fmt.Errorf("looking up accounts: %w", err)
	}
	result := make(map[tb.Uint128]int64, len(accounts))
	for _, a := range accounts {
		result[a.ID] = uint128ToInt64(a.CreditsPosted) - uint128ToInt64(a.DebitsPosted)
	}
	return result, nil
}
