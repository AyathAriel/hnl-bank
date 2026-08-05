// Package mcptools define las herramientas bancarias expuestas por el servidor
// MCP (cmd/mcpserver). Cada tool delega en internal/banking, la misma capa de
// dominio usada por los endpoints REST, y todas están ancladas a un único
// usuario autenticado (el servidor MCP se lanza por sesión de chat).
package mcptools

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hnl/bank-backend/internal/banking"
	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/models"
)

// Register añade todas las tools bancarias al servidor MCP, ancladas a userID.
func Register(server *mcp.Server, svc *banking.Service, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_accounts",
		Description: "Lista todas las cuentas bancarias del usuario autenticado, con su número, tipo y saldo actual.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListAccountsInput) (*mcp.CallToolResult, ListAccountsOutput, error) {
		accounts, err := svc.GetAccounts(ctx, userID)
		if err != nil {
			return nil, ListAccountsOutput{}, err
		}
		return nil, ListAccountsOutput{Accounts: toAccountViews(accounts)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_balance",
		Description: "Obtiene el saldo de una cuenta específica del usuario (o el total de todas sus cuentas si no se especifica número de cuenta).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GetBalanceInput) (*mcp.CallToolResult, GetBalanceOutput, error) {
		accounts, err := svc.GetAccounts(ctx, userID)
		if err != nil {
			return nil, GetBalanceOutput{}, err
		}
		if input.AccountNumber == "" {
			total := int64(0)
			for _, a := range accounts {
				total += a.BalanceCents
			}
			return nil, GetBalanceOutput{Accounts: toAccountViews(accounts), TotalBalance: ledger.CentsToDecimalString(total)}, nil
		}
		for _, a := range accounts {
			if a.AccountNumber == input.AccountNumber {
				return nil, GetBalanceOutput{Accounts: []AccountView{toAccountView(a)}}, nil
			}
		}
		return nil, GetBalanceOutput{}, fmt.Errorf("no tienes ninguna cuenta con número %s", input.AccountNumber)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transaction_history",
		Description: "Obtiene las transacciones recientes del usuario, opcionalmente filtradas por número de cuenta.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input HistoryInput) (*mcp.CallToolResult, HistoryOutput, error) {
		limit := input.Limit
		if limit <= 0 || limit > 50 {
			limit = 10
		}
		txs, _, err := svc.GetHistory(ctx, userID, input.AccountNumber, 1, limit)
		if err != nil {
			return nil, HistoryOutput{}, err
		}
		return nil, HistoryOutput{Transactions: txs}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "deposit",
		Description: "Deposita fondos en una cuenta del usuario. Es una acción CRÍTICA: si 'confirmed' no es true, " +
			"la tool NO ejecuta el depósito y responde pidiendo confirmación explícita al usuario en lenguaje natural " +
			"antes de volver a llamarla con confirmed=true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input DepositInput) (*mcp.CallToolResult, ActionOutput, error) {
		if !input.Confirmed {
			return nil, needsConfirmation(fmt.Sprintf("Vas a depositar $%s en la cuenta %s. ¿Confirmas la operación?", input.Amount, input.AccountNumber)), nil
		}
		account, err := svc.Deposit(ctx, userID, input.AccountNumber, input.Amount, input.Description)
		if err != nil {
			return nil, errorOutput(err), nil
		}
		view := toAccountView(account)
		return nil, ActionOutput{Status: "completed", Message: "Depósito realizado con éxito.", Account: &view}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "withdraw",
		Description: "Retira fondos de una cuenta del usuario. Es una acción CRÍTICA: si 'confirmed' no es true, " +
			"la tool NO ejecuta el retiro y responde pidiendo confirmación explícita al usuario en lenguaje natural " +
			"antes de volver a llamarla con confirmed=true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input WithdrawInput) (*mcp.CallToolResult, ActionOutput, error) {
		if !input.Confirmed {
			return nil, needsConfirmation(fmt.Sprintf("Vas a retirar $%s de la cuenta %s. ¿Confirmas la operación?", input.Amount, input.AccountNumber)), nil
		}
		account, err := svc.Withdraw(ctx, userID, input.AccountNumber, input.Amount, input.Description)
		if err != nil {
			return nil, errorOutput(err), nil
		}
		view := toAccountView(account)
		return nil, ActionOutput{Status: "completed", Message: "Retiro realizado con éxito.", Account: &view}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "transfer",
		Description: "Transfiere fondos desde una cuenta del usuario hacia otra cuenta (propia o de un tercero). " +
			"Es una acción CRÍTICA: si 'confirmed' no es true, la tool NO ejecuta la transferencia y responde " +
			"pidiendo confirmación explícita al usuario en lenguaje natural antes de volver a llamarla con confirmed=true.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input TransferInput) (*mcp.CallToolResult, ActionOutput, error) {
		if !input.Confirmed {
			return nil, needsConfirmation(fmt.Sprintf("Vas a transferir $%s desde la cuenta %s hacia la cuenta %s. ¿Confirmas la operación?", input.Amount, input.FromAccountNumber, input.ToAccountNumber)), nil
		}
		account, err := svc.Transfer(ctx, userID, input.FromAccountNumber, input.ToAccountNumber, input.Amount, input.Description)
		if err != nil {
			return nil, errorOutput(err), nil
		}
		view := toAccountView(account)
		return nil, ActionOutput{Status: "completed", Message: "Transferencia realizada con éxito.", Account: &view}, nil
	})
}

func needsConfirmation(message string) ActionOutput {
	return ActionOutput{Status: "needs_confirmation", Message: message}
}

func errorOutput(err error) ActionOutput {
	msg := err.Error()
	switch {
	case errors.Is(err, banking.ErrInsufficientFunds):
		msg = "Fondos insuficientes para completar la operación."
	case errors.Is(err, banking.ErrAccountNotFound), errors.Is(err, banking.ErrDestinationNotFound):
		msg = "No se encontró la cuenta indicada."
	case errors.Is(err, banking.ErrForbidden):
		msg = "Esa cuenta no pertenece al usuario autenticado."
	case errors.Is(err, banking.ErrSameAccount):
		msg = "No se puede transferir a la misma cuenta."
	}
	return ActionOutput{Status: "error", Message: msg}
}

func toAccountViews(accounts []models.Account) []AccountView {
	views := make([]AccountView, 0, len(accounts))
	for _, a := range accounts {
		views = append(views, toAccountView(a))
	}
	return views
}

func toAccountView(a models.Account) AccountView {
	return AccountView{
		AccountNumber: a.AccountNumber,
		AccountType:   string(a.AccountType),
		Currency:      a.Currency,
		Balance:       a.Balance,
	}
}
