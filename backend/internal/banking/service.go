// Package banking contiene la lógica de dominio bancaria (registro, cuentas,
// depósitos, retiros, transferencias, historial). Es la única capa de negocio
// del sistema: tanto los handlers REST (internal/httpapi) como las tools MCP
// (internal/mcptools) delegan aquí, evitando duplicar validaciones.
package banking

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tb "github.com/tigerbeetle/tigerbeetle-go"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/models"
)

var (
	ErrEmailTaken          = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountNotFound     = errors.New("account not found")
	ErrForbidden           = errors.New("account does not belong to user")
	ErrSameAccount         = errors.New("cannot transfer to the same account")
	ErrDestinationNotFound = errors.New("destination account not found")
	ErrInsufficientFunds   = errors.New("insufficient funds")
)

type Service struct {
	pool   *pgxpool.Pool
	ledger *ledger.Client
	jwtSecret string
	jwtExpiryHours int
}

func NewService(pool *pgxpool.Pool, ledgerClient *ledger.Client, jwtSecret string, jwtExpiryHours int) *Service {
	return &Service{pool: pool, ledger: ledgerClient, jwtSecret: jwtSecret, jwtExpiryHours: jwtExpiryHours}
}

// Register crea un usuario nuevo con una cuenta checking asociada.
func (s *Service) Register(ctx context.Context, email, password, fullName string) (models.User, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
		return models.User{}, "", fmt.Errorf("checking existing email: %w", err)
	}
	if exists {
		return models.User{}, "", ErrEmailTaken
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return models.User{}, "", fmt.Errorf("hashing password: %w", err)
	}

	userID := uuid.NewString()
	now := time.Now()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.User{}, "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, full_name, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$5)`,
		userID, email, hash, fullName, now,
	); err != nil {
		return models.User{}, "", fmt.Errorf("inserting user: %w", err)
	}

	accountNumber, err := s.generateUniqueAccountNumber(ctx, tx)
	if err != nil {
		return models.User{}, "", err
	}

	tbAccountID := ledger.NewCustomerAccountID()
	if err := s.ledger.CreateCustomerAccounts([]struct {
		ID          tb.Uint128
		AccountType string
	}{{ID: tbAccountID, AccountType: "checking"}}); err != nil {
		return models.User{}, "", fmt.Errorf("creating tigerbeetle account: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO accounts (id, account_number, user_id, tigerbeetle_account_id, currency, account_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uuid.NewString(), accountNumber, userID, tbAccountID.BigInt().String(), "USD", "checking", now,
	); err != nil {
		return models.User{}, "", fmt.Errorf("inserting account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.User{}, "", fmt.Errorf("commit tx: %w", err)
	}

	token, _, err := auth.GenerateToken(s.jwtSecret, userID, email, s.jwtExpiryHours)
	if err != nil {
		return models.User{}, "", fmt.Errorf("generating token: %w", err)
	}

	user := models.User{ID: userID, Email: email, FullName: fullName, CreatedAt: now}
	return user, token, nil
}

func (s *Service) generateUniqueAccountNumber(ctx context.Context, tx pgx.Tx) (string, error) {
	for i := 0; i < 10; i++ {
		number := fmt.Sprintf("4001-%04d-%04d-%04d", rand.Intn(10000), rand.Intn(10000), rand.Intn(10000))
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE account_number = $1)`, number).Scan(&exists); err != nil {
			return "", fmt.Errorf("checking account number: %w", err)
		}
		if !exists {
			return number, nil
		}
	}
	return "", fmt.Errorf("could not generate a unique account number")
}

// LoginResult representa el resultado de un intento de login. Si el usuario
// tiene 2FA activado, RequiresTOTP viene en true y Token queda vacío: el
// cliente debe canjear PendingToken + un código válido en
// POST /api/auth/2fa/verify para obtener una sesión real.
type LoginResult struct {
	User         models.User
	Token        string
	RequiresTOTP bool
	PendingToken string
}

// Login valida credenciales y devuelve una sesión nueva, o un token pendiente
// de segundo factor si el usuario tiene 2FA activado.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	var user models.User
	var hash string
	err := s.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, full_name, created_at, totp_enabled FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &hash, &user.FullName, &user.CreatedAt, &user.TOTPEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("querying user: %w", err)
	}

	if !auth.CheckPassword(hash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}

	if user.TOTPEnabled {
		pending, err := auth.GeneratePending2FAToken(s.jwtSecret, user.ID, user.Email)
		if err != nil {
			return LoginResult{}, fmt.Errorf("generating pending token: %w", err)
		}
		return LoginResult{User: user, RequiresTOTP: true, PendingToken: pending}, nil
	}

	token, _, err := auth.GenerateToken(s.jwtSecret, user.ID, user.Email, s.jwtExpiryHours)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generating token: %w", err)
	}
	return LoginResult{User: user, Token: token}, nil
}
