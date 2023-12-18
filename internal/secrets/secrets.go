package secrets

import (
	"os"
)

type Secrets interface {
	DatabaseUrl() string
	CognitoClientId() string
}

func New() Secrets {
	return &secrets{
		databaseUrl:     os.Getenv("DATABASE_URL"),
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
