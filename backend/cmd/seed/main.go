package main

import (
	"context"
	"log"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/config"
	"github.com/eakillidev/Care-Flow/backend/internal/database"
	"github.com/eakillidev/Care-Flow/backend/internal/seed"
)

func main() {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := seed.Development(ctx, pool); err != nil {
		log.Fatalf("seed development data: %v", err)
	}
	log.Print("development seed completed")
}
