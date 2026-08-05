// Package models define las estructuras de dominio compartidas entre capas.
package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type AccountType string

const (
	AccountTypeChecking   AccountType = "checking"
	AccountTypeSavings    AccountType = "savings"
	AccountTypeInvestment AccountType = "investment"
)

type Account struct {
	ID                  string      `json:"id"`
	AccountNumber       string      `json:"account_number"`
	UserID              string      `json:"user_id"`
	TigerBeetleID       string      `json:"-"`
	Currency            string      `json:"currency"`
	AccountType         AccountType `json:"account_type"`
	CreatedAt           time.Time   `json:"created_at"`
	BalanceCents        int64       `json:"balance_cents"`
	Balance             string      `json:"balance"`
}

type TransactionType string

const (
	TransactionTypeDeposit         TransactionType = "deposit"
	TransactionTypeWithdrawal      TransactionType = "withdrawal"
	TransactionTypeTransfer        TransactionType = "transfer"
	TransactionTypeInternalTransfer TransactionType = "internal_transfer"
)

type TransactionStatus string

const (
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

type Transaction struct {
	ID                  string            `json:"id"`
	TigerBeetleTransferID *string         `json:"-"`
	FromAccountNumber   string            `json:"from_account_number"`
	ToAccountNumber     string            `json:"to_account_number"`
	Amount              string            `json:"amount"`
	Type                TransactionType   `json:"type"`
	Description         string            `json:"description"`
	Status              TransactionStatus `json:"status"`
	UserID              *string           `json:"-"`
	CreatedAt           time.Time         `json:"created_at"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}
