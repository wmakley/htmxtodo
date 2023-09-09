package main

import (
	"database/sql"
	"embed"
	"github.com/joho/godotenv"
	"htmxtodo/internal/app"
	"htmxtodo/internal/repo"
	"log"
	"os"
)

//go:embed all:views/*
var viewsFS embed.FS

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

	r := repo.New(db)

	config := app.NewConfigFromEnvironment(viewsFS, r)

	a := app.New(&config)

	log.Fatal(a.Listen(config.Host + ":" + config.Port))
}