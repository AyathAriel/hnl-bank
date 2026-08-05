// Package db maneja la conexión a PostgreSQL y la ejecución de migraciones.
package db

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect abre el pool de conexiones, reintentando mientras Postgres arranca
// (relevante en docker-compose, donde el healthcheck puede tardar unos segundos).
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	deadline := time.Now().Add(60 * time.Second)
	for {
		pool, err = pgxpool.New(ctx, databaseURL)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			pingErr := pool.Ping(pingCtx)
			cancel()
			if pingErr == nil {
				return pool, nil
			}
			pool.Close()
			err = pingErr
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("could not connect to postgres after retries: %w", err)
		}
		log.Printf("waiting for postgres: %v", err)
		time.Sleep(2 * time.Second)
	}
}

// Migrate ejecuta todos los archivos .sql embebidos en orden alfabético.
// Cada migración debe ser idempotente (CREATE TABLE IF NOT EXISTS, etc.),
// suficiente para el alcance de esta prueba técnica.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("applying migration %s: %w", name, err)
		}
		log.Printf("applied migration %s", name)
	}

	return nil
}
