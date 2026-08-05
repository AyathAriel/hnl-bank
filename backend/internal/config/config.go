// Package config carga la configuración del servicio desde variables de entorno.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	TBClusterID      uint64
	TBAddress        string
	JWTSecret        string
	JWTExpiryHours   int
	AnthropicAPIKey  string
	AnthropicModel   string
	CORSOrigin       string
	MCPServerBinPath string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		TBAddress:        getEnv("TB_ADDRESS", "127.0.0.1:3000"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AnthropicAPIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:   getEnv("ANTHROPIC_MODEL", "claude-sonnet-5"),
		CORSOrigin:       getEnv("CORS_ORIGIN", "http://localhost:5173"),
		MCPServerBinPath: getEnv("MCP_SERVER_BIN", "/app/mcpserver"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	clusterID, err := strconv.ParseUint(getEnv("TB_CLUSTER_ID", "0"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid TB_CLUSTER_ID: %w", err)
	}
	cfg.TBClusterID = clusterID

	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}
	cfg.JWTExpiryHours = expiryHours

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
