package config

import (
	"database/sql"
	"embed"
	"htmxtodo/internal/constants"
	"htmxtodo/internal/repo"
	"htmxtodo/internal/secrets"
	"net/http"
	"os"
)

// Config is the global config for the app router. Host and Port are needed for absolute URL generation.
type Config struct {
	Env              string
	Host             string
	Port             string
	Repo             repo.Repository
	CookieSecure     bool
	DisableLogColors bool
	EnableStackTrace bool
	StaticFS         http.FileSystem
	Secrets          secrets.Secrets
}

func NewConfigFromEnvironment(dbConn *sql.DB, staticFS *embed.FS) *Config {
	env := os.Getenv("ENV")

	return &Config{
		Env:              env,
		Host:             os.Getenv("HOST"),
		Port:             os.Getenv("PORT"),
		Repo:             repo.New(dbConn),
		CookieSecure:     env == constants.EnvProduction,
		DisableLogColors: env == constants.EnvProduction,
		EnableStackTrace: env == constants.EnvDevelopment,
		StaticFS:         http.FS(staticFS),
		Secrets:          secrets.New(),
	}
}

func NewTestConfig(dbConn *sql.DB) *Config {
	return &Config{
		Env:              constants.EnvTest,
		Host:             os.Getenv("HOST"),
		Port:             os.Getenv("PORT"),
		Repo:             repo.New(dbConn),
		CookieSecure:     false,
		DisableLogColors: false,
		EnableStackTrace: true,
		StaticFS:         http.Dir("./static"),
		Secrets:          secrets.New(),
	}
}
