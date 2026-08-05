// Package seed carga el dataset de prueba (backend/seed-data/datos-prueba-HNL.json)
// en PostgreSQL y TigerBeetle. Es idempotente: si ya hay usuarios en la base,
// no hace nada (para no duplicar datos en reinicios de docker-compose).
//
// Decisión de diseño (documentada también en el README): `initial_balance` se
// interpreta como el saldo ACTUAL de cada cuenta y se carga en TigerBeetle
// mediante una única transferencia de apertura (EXTERNAL -> cuenta). Las
// transacciones históricas del JSON se insertan en PostgreSQL como registro de
// auditoría/historial para la UI, pero no se re-ejecutan contra TigerBeetle:
// el dataset sintético no garantiza que initial_balance sea el resultado
// contable exacto de reproducir ese historial, y forzarlo produciría rechazos
// de sobregiro arbitrarios contra un ledger que sí exige partida doble real.
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tb "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/ledger"
)

const bcryptWorkers = 16

// Run ejecuta el seeding si la tabla users está vacía.
func Run(ctx context.Context, pool *pgxpool.Pool, ledgerClient *ledger.Client, datasetPath string) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("checking existing users: %w", err)
	}
	if count > 0 {
		log.Printf("seed: users table is not empty (%d rows), skipping seed", count)
		return nil
	}

	data, err := os.ReadFile(datasetPath)
	if err != nil {
		return fmt.Errorf("reading dataset %s: %w", datasetPath, err)
	}
	var ds Dataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return fmt.Errorf("parsing dataset: %w", err)
	}
	log.Printf("seed: loaded dataset with %d users, %d accounts, %d transactions", len(ds.Users), len(ds.Accounts), len(ds.Transactions))

	if err := ledgerClient.EnsureExternalAccount(); err != nil {
		return fmt.Errorf("ensuring external account: %w", err)
	}

	survivingUserIDs, err := seedUsers(ctx, pool, ds.Users)
	if err != nil {
		return fmt.Errorf("seeding users: %w", err)
	}

	accountToUser, err := seedAccounts(ctx, pool, ledgerClient, ds.Accounts, survivingUserIDs)
	if err != nil {
		return fmt.Errorf("seeding accounts: %w", err)
	}

	if err := seedTransactions(ctx, pool, ds.Transactions, accountToUser); err != nil {
		return fmt.Errorf("seeding transactions: %w", err)
	}

	log.Printf("seed: completed successfully")
	return nil
}

// seedUsers inserta los usuarios del dataset y devuelve el conjunto de IDs
// efectivamente insertados. El dataset sintético contiene un puñado de emails
// duplicados entre usuarios distintos (email debe ser único para el login):
// se conserva la primera aparición de cada email y se descartan las
// siguientes, junto con sus cuentas/movimientos asociados más adelante.
func seedUsers(ctx context.Context, pool *pgxpool.Pool, users []SeedUser) (map[string]bool, error) {
	survivingIDs := make(map[string]bool, len(users))
	if len(users) == 0 {
		return survivingIDs, nil
	}

	seenEmails := make(map[string]bool, len(users))
	unique := make([]SeedUser, 0, len(users))
	skipped := 0
	for _, u := range users {
		email := strings.ToLower(strings.TrimSpace(u.Email))
		if seenEmails[email] {
			skipped++
			continue
		}
		seenEmails[email] = true
		unique = append(unique, u)
	}
	if skipped > 0 {
		log.Printf("seed: skipped %d user(s) with duplicate email", skipped)
	}

	hashes := make([]string, len(unique))
	var wg sync.WaitGroup
	jobs := make(chan int)

	for w := 0; w < bcryptWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				h, err := auth.HashPassword(unique[i].Password)
				if err != nil {
					h = "" // se detecta más abajo si hace falta; no debería fallar con bcrypt
				}
				hashes[i] = h
			}
		}()
	}
	for i := range unique {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	rows := make([][]interface{}, 0, len(unique))
	for i, u := range unique {
		if hashes[i] == "" {
			return nil, fmt.Errorf("failed to hash password for user %s", u.Email)
		}
		createdAt := u.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		rows = append(rows, []interface{}{u.ID, strings.ToLower(strings.TrimSpace(u.Email)), hashes[i], u.FullName, createdAt, createdAt})
		survivingIDs[u.ID] = true
	}

	_, err := pool.CopyFrom(ctx,
		pgx.Identifier{"users"},
		[]string{"id", "email", "password_hash", "full_name", "created_at", "updated_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, err
	}
	return survivingIDs, nil
}

func seedAccounts(ctx context.Context, pool *pgxpool.Pool, ledgerClient *ledger.Client, allAccounts []SeedAccount, survivingUserIDs map[string]bool) (map[string]string, error) {
	accountToUser := make(map[string]string, len(allAccounts))

	accounts := make([]SeedAccount, 0, len(allAccounts))
	skipped := 0
	for _, a := range allAccounts {
		if !survivingUserIDs[a.UserID] {
			skipped++
			continue
		}
		accounts = append(accounts, a)
	}
	if skipped > 0 {
		log.Printf("seed: skipped %d account(s) belonging to a discarded duplicate user", skipped)
	}
	if len(accounts) == 0 {
		return accountToUser, nil
	}

	type generated struct {
		tbID tb.Uint128
	}
	gen := make([]generated, len(accounts))

	tbAccounts := make([]struct {
		ID          tb.Uint128
		AccountType string
	}, 0, len(accounts))

	for i, a := range accounts {
		id := ledger.NewCustomerAccountID()
		gen[i] = generated{tbID: id}
		tbAccounts = append(tbAccounts, struct {
			ID          tb.Uint128
			AccountType string
		}{ID: id, AccountType: a.AccountType})
	}

	if err := ledgerClient.CreateCustomerAccounts(tbAccounts); err != nil {
		return nil, fmt.Errorf("creating tigerbeetle accounts: %w", err)
	}

	rows := make([][]interface{}, 0, len(accounts))
	openings := make([]ledger.OpeningBalanceTransfer, 0, len(accounts))
	now := time.Now()

	for i, a := range accounts {
		accountID := newUUID()
		rows = append(rows, []interface{}{
			accountID, a.AccountNumber, a.UserID, gen[i].tbID.BigInt().String(), a.Currency, a.AccountType, now,
		})
		accountToUser[a.AccountNumber] = a.UserID

		cents := ledger.FloatToCents(a.InitialBalance)
		if cents > 0 {
			openings = append(openings, ledger.OpeningBalanceTransfer{AccountID: gen[i].tbID, AmountCents: cents})
		}
	}

	if _, err := pool.CopyFrom(ctx,
		pgx.Identifier{"accounts"},
		[]string{"id", "account_number", "user_id", "tigerbeetle_account_id", "currency", "account_type", "created_at"},
		pgx.CopyFromRows(rows),
	); err != nil {
		return nil, fmt.Errorf("inserting accounts: %w", err)
	}

	if err := ledgerClient.CreateOpeningBalances(openings); err != nil {
		return nil, fmt.Errorf("creating opening balances: %w", err)
	}

	return accountToUser, nil
}

func seedTransactions(ctx context.Context, pool *pgxpool.Pool, transactions []SeedTransaction, accountToUser map[string]string) error {
	if len(transactions) == 0 {
		return nil
	}

	rows := make([][]interface{}, 0, len(transactions))
	for _, t := range transactions {
		var userID interface{}
		if owner, ok := accountToUser[t.FromAccount]; ok {
			userID = owner
		} else if owner, ok := accountToUser[t.ToAccount]; ok {
			userID = owner
		} else {
			userID = nil
		}

		status := t.Status
		if status == "" {
			status = "completed"
		}
		txType := t.Type
		if txType == "" {
			txType = "transfer"
		}

		rows = append(rows, []interface{}{
			newUUID(), nil, t.FromAccount, t.ToAccount,
			fmt.Sprintf("%.2f", t.Amount), txType, t.Description, status, userID, t.Timestamp,
		})
	}

	_, err := pool.CopyFrom(ctx,
		pgx.Identifier{"transactions"},
		[]string{"id", "tigerbeetle_transfer_id", "from_account_number", "to_account_number", "amount", "type", "description", "status", "user_id", "created_at"},
		pgx.CopyFromRows(rows),
	)
	return err
}
