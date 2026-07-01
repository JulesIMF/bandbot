package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/julesimf/bandbot/internal/bot"
	"github.com/julesimf/bandbot/internal/storage"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://bandbot:bandbot@localhost:5432/bandbot?sslmode=disable"
	}

	ctx := context.Background()

	store, err := storage.NewPostgres(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()
	log.Println("Connected to database")

	b, err := bot.New(token, store)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	go b.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("Shutting down...")
	b.Stop()
}
