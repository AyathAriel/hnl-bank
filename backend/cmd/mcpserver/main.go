// cmd/mcpserver es un servidor MCP real (stdio, JSON-RPC) que expone las
// operaciones bancarias como tools. Se lanza como subproceso independiente,
// uno por sesión de chat, recibiendo el ID del usuario autenticado por flag
// para que todas las tools operen exclusivamente sobre sus propias cuentas.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hnl/bank-backend/internal/banking"
	"github.com/hnl/bank-backend/internal/config"
	"github.com/hnl/bank-backend/internal/db"
	"github.com/hnl/bank-backend/internal/ledger"
	"github.com/hnl/bank-backend/internal/mcptools"
)

func main() {
	userID := flag.String("user-id", "", "ID del usuario autenticado propietario de esta sesión de chat")
	flag.Parse()

	if *userID == "" {
		log.Fatal("mcpserver: --user-id is required")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("mcpserver: loading config: %v", err)
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("mcpserver: connecting to postgres: %v", err)
	}
	defer pool.Close()

	ledgerClient, err := ledger.NewClient(cfg.TBClusterID, []string{cfg.TBAddress})
	if err != nil {
		log.Fatalf("mcpserver: connecting to tigerbeetle: %v", err)
	}
	defer ledgerClient.Close()

	bankingService := banking.NewService(pool, ledgerClient, cfg.JWTSecret, cfg.JWTExpiryHours)

	server := mcp.NewServer(&mcp.Implementation{Name: "hnl-bank", Version: "1.0.0"}, nil)
	mcptools.Register(server, bankingService, *userID)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcpserver: %v", err)
	}
}
