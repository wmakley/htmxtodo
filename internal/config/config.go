package config

import (
	"database/sql"
	"embed"
	"htmxtodo/internal/constants"
	"htmxtodo/internal/repo"
	"os"
)

// Config is the global config for the app router. Host and Port are needed for absolute URL generation.
type Config struct {
	Env              string
	Host             string
	Port             string
	Repo             repo.Repository
	CognitoClientId  string
	CookieSecure     bool
	DatabaseUrl      string
	DisableLogColors bool
	EnableStackTrace bool
	StaticFS         embed.FS
}

func NewConfigFromEnvironment(dbConn *sql.DB, staticFS embed.FS) Config {
	env := os.Getenv("ENV")
	dbUrlKey := "DATABASE_URL"
	if env == "TEST" {
		dbUrlKey = "TEST_DATABASE_URL"
	}

	return Config{
		Env:              env,
		Host:             os.Getenv("HOST"),
		Port:             os.Getenv("PORT"),
		Repo:             repo.New(dbConn),
		CognitoClientId:  os.Getenv("COGNITO_CLIENT_ID"),
		CookieSecure:     env == constants.EnvProduction,
		DatabaseUrl:      os.Getenv(dbUrlKey),
		DisableLogColors: env == constants.EnvProduction,
		EnableStackTrace: env == constants.EnvDevelopment,
		StaticFS:         staticFS,
	}
}
