package components

import (
	"context"
	"htmxtodo/internal/constants"
)

func GetCsrfToken(ctx context.Context) string {
	if csrfToken, ok := ctx.Value(constants.CsrfTokenContextKey).(string); ok {
		return csrfToken
	}
	return ""
}

func GetLoggedIn(ctx context.Context) bool {
	if loggedIn, ok := ctx.Value(constants.LoggedInSessionKey).(bool); ok {
		return loggedIn
	}
	return false
}
