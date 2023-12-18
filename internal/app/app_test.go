package app

import (
	"database/sql"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"htmxtodo/internal/config"
	"log"
	"net/http/httptest"
	"os"
	"testing"
)

var testApp *fiber.App

func TestMain(m *testing.M) {
	err := godotenv.Load("../../.env.test")
	if err != nil {
		log.Fatalf("Error loading .env.test file: %s", err.Error())
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

	cfg := config.NewTestConfig(db)
	testApp = New(cfg)

	os.Exit(m.Run())
}

func TestLogin(t *testing.T) {
	req := httptest.NewRequest("GET", "/login", nil)
	resp, _ := testApp.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatal("response was not 200, was ", resp.Status)
	}
}
