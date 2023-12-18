package config

import (
	"database/sql"
	"embed"
	"htmxtodo/internal/constants"
	"htmxtodo/internal/repo"
	"htmxtodo/internal/secrets"
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
	StaticFS         embed.FS
	Secrets          secrets.Secrets
}

func NewConfigFromEnvironment(dbConn *sql.DB, staticFS embed.FS) Config {
	env := os.Getenv("ENV")

	return Config{
		Env:              env,
		Host:             os.Getenv("HOST"),
		Port:             os.Getenv("PORT"),
		Repo:             repo.New(dbConn),
		CookieSecure:     env == constants.EnvProduction,
		DisableLogColors: env == constants.EnvProduction,
		EnableStackTrace: env == constants.EnvDevelopment,
		StaticFS:         staticFS,
		Secrets:          secrets.New(env),
	}
}
