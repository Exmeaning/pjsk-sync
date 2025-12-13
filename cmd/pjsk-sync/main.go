package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pjsk-sync/internal/config"
	"pjsk-sync/internal/db"
	"pjsk-sync/internal/sync"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Open(ctx, cfg.PostgresConnString, cfg.PGSSLMode)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	if err := sync.Run(ctx, pool, cfg); err != nil {
		log.Fatalf("sync run: %v", err)
	}

	log.Printf("done")
}