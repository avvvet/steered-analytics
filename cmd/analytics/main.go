package main

import (
	"log"
	"net/http"
	"os"

	"github.com/avvvet/steered-analytics/internal/analytics"
	"github.com/joho/godotenv"
	bolt "go.etcd.io/bbolt"
)

func main() {
	godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./analytics.db"
	}

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	store := analytics.NewStore(db)
	if err := store.Init(); err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	telegram := analytics.NewTelegram(
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		os.Getenv("TELEGRAM_CHAT_ID"),
	)
	telegram.Verify()

	srv := analytics.NewServer(store, telegram, os.Getenv("API_TOKEN"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("analytics server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
