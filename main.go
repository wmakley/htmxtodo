package main

import (
	"database/sql"
	"embed"
	"github.com/joho/godotenv"
	"htmxtodo/internal/app"
	"htmxtodo/internal/config"
	"log"
	"os"
)

//go:embed static/*
var staticEmbedFS embed.FS

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("failed to close db: %+v", err)
		}
	}(db)

	cfg := config.NewConfigFromEnvironment(db, &staticEmbedFS)

	a := app.New(cfg)

	log.Fatal(a.Listen(cfg.Host + ":" + cfg.Port))
}
