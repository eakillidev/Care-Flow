package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/eakillidev/Care-Flow/backend/internal/config"
	"github.com/eakillidev/Care-Flow/backend/internal/database"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	directory := flag.String("path", "migrations", "directory containing migration files")
	flag.Parse()

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, *directory, *direction); err != nil {
		log.Fatalf("run %s migrations: %v", *direction, err)
	}
	log.Printf("%s migrations completed", *direction)
}
