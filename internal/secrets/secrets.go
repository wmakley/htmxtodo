package secrets

import (
	"htmxtodo/internal/constants"
	"os"
)

type Secrets interface {
	DatabaseUrl() string
	CognitoClientId() string
}

func New(env string) Secrets {
	dbUrlKey := "DATABASE_URL"
	if env == constants.EnvTest {
		dbUrlKey = "TEST_DATABASE_URL"
	}

	return &secrets{
		databaseUrl:     os.Getenv(dbUrlKey),
		cognitoClientId: os.Getenv("COGNITO_CLIENT_ID"),
	}
}

type secrets struct {
	databaseUrl     string
	cognitoClientId string
}

func (s secrets) DatabaseUrl() string {
	return s.databaseUrl
}

func (s secrets) CognitoClientId() string {
	return s.cognitoClientId
}
