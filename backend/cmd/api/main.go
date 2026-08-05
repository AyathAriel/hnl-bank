// cmd/api es el servidor HTTP principal: expone la API REST y, si hay una
// ANTHROPIC_API_KEY configurada, también el endpoint de chat con IA.
// Al arrancar: conecta a PostgreSQL y TigerBeetle (con reintentos, para
// tolerar el orden de arranque de docker-compose), aplica migraciones y
// siembra el dataset de prueba si la base está vacía.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/hnl/bank-backend/internal/auth"
	"github.com/hnl/bank-backend/internal/banking"
	"github.com/hnl/bank-backend/internal/chat"
	"github.com/hnl/bank-backend/internal/config"
	"github.com/hnl/bank-backend/internal/db"
	"github.com/hnl/bank-backend/internal/httpapi"
	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/seed"
	"github.com/hnl/bank-backend/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to postgres: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("running migrations: %v", err)
	}

	ledgerClient, err := connectLedgerWithRetry(cfg.TBClusterID, []string{cfg.TBAddress})
	if err != nil {
		log.Fatalf("connecting to tigerbeetle: %v", err)
	}
	defer ledgerClient.Close()

	if err := ledgerClient.EnsureExternalAccount(); err != nil {
		log.Fatalf("ensuring external account: %v", err)
	}

	if err := seed.Run(ctx, pool, ledgerClient, "seed-data/datos-prueba-HNL.json"); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	bankingService := banking.NewService(pool, ledgerClient, cfg.JWTSecret, cfg.JWTExpiryHours)
	revocationStore := auth.NewRevocationStore(pool)

	var chatService *chat.Service
	if cfg.AnthropicAPIKey != "" {
		anthropicClient := chat.NewAnthropicClient(cfg.AnthropicAPIKey, cfg.AnthropicModel)
		chatService = chat.NewService(anthropicClient, cfg.MCPServerBinPath)
		log.Printf("chat: enabled (model=%s)", cfg.AnthropicModel)
	} else {
		log.Printf("chat: disabled (ANTHROPIC_API_KEY not set)")
	}

	hub := ws.NewHub()

	var server *httpapi.Server
	if chatService != nil {
		server = httpapi.NewServer(cfg, bankingService, revocationStore, chatService, hub)
	} else {
		server = httpapi.NewServer(cfg, bankingService, revocationStore, nil, hub)
	}

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, server.Router()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

func connectLedgerWithRetry(clusterID uint64, addresses []string) (*ledger.Client, error) {
	var lastErr error
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		client, err := ledger.NewClient(clusterID, addresses)
		if err == nil {
			return client, nil
		}
		lastErr = err
		log.Printf("waiting for tigerbeetle: %v", err)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}
