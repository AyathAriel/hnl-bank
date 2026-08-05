package mcptools

import "github.com/hnl/bank-backend/internal/models"

type AccountView struct {
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	Currency      string `json:"currency"`
	Balance       string `json:"balance"`
}

type ListAccountsInput struct{}

type ListAccountsOutput struct {
	Accounts []AccountView `json:"accounts"`
}

type GetBalanceInput struct {
	AccountNumber string `json:"account_number,omitempty" jsonschema:"número de cuenta a consultar; si se omite, se devuelven todas las cuentas del usuario"`
}

type GetBalanceOutput struct {
	Accounts     []AccountView `json:"accounts"`
	TotalBalance string        `json:"total_balance,omitempty"`
}

type HistoryInput struct {
	AccountNumber string `json:"account_number,omitempty" jsonschema:"número de cuenta a filtrar; si se omite, se devuelven movimientos de todas las cuentas del usuario"`
	Limit         int    `json:"limit,omitempty" jsonschema:"cantidad máxima de transacciones a devolver (por defecto 10, máximo 50)"`
}

type HistoryOutput struct {
	Transactions []models.Transaction `json:"transactions"`
}

type DepositInput struct {
	AccountNumber string `json:"account_number" jsonschema:"número de cuenta destino del depósito"`
	Amount        string `json:"amount" jsonschema:"monto a depositar, como decimal, ej. '100.50'"`
	Description   string `json:"description,omitempty" jsonschema:"descripción opcional del depósito"`
	Confirmed     bool   `json:"confirmed,omitempty" jsonschema:"debe ser true solo después de que el usuario confirmó explícitamente la operación en el chat"`
}

type WithdrawInput struct {
	AccountNumber string `json:"account_number" jsonschema:"número de cuenta de la que se retiran los fondos"`
	Amount        string `json:"amount" jsonschema:"monto a retirar, como decimal, ej. '100.50'"`
	Description   string `json:"description,omitempty" jsonschema:"descripción opcional del retiro"`
	Confirmed     bool   `json:"confirmed,omitempty" jsonschema:"debe ser true solo después de que el usuario confirmó explícitamente la operación en el chat"`
}

type TransferInput struct {
	FromAccountNumber string `json:"from_account_number" jsonschema:"número de cuenta de origen (debe pertenecer al usuario)"`
	ToAccountNumber   string `json:"to_account_number" jsonschema:"número de cuenta destino"`
	Amount            string `json:"amount" jsonschema:"monto a transferir, como decimal, ej. '100.50'"`
	Description       string `json:"description,omitempty" jsonschema:"descripción opcional de la transferencia"`
	Confirmed         bool   `json:"confirmed,omitempty" jsonschema:"debe ser true solo después de que el usuario confirmó explícitamente la operación en el chat"`
}

// ActionOutput es la salida común de deposit/withdraw/transfer.
type ActionOutput struct {
	Status  string       `json:"status"` // needs_confirmation | completed | error
	Message string       `json:"message"`
	Account *AccountView `json:"account,omitempty"`
}
