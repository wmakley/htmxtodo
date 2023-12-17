package app

import (
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/session"
	"htmxtodo/internal/constants"
)

func SetLoggedIn(sessionStore *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sess, err := sessionStore.Get(c)
		if err != nil {
			panic(err)
		}

		loggedIn := sess.Get(constants.LoggedInSessionKey) == "true"
		c.Locals(constants.LoggedInSessionKey, loggedIn)

		return c.Next()
	}
}

func RequireLoggedIn(c *fiber.Ctx) error {
	loggedIn := c.Locals(constants.LoggedInSessionKey).(bool)
	if !loggedIn {
		fiberlog.Error("not logged in, redirecting to login")
		return c.Redirect("/login", fiber.StatusFound)
	}

	c.Locals(constants.LoggedInSessionKey, loggedIn)

	return c.Next()
}

func RedirectInternalIfLoggedIn(c *fiber.Ctx) error {
	loggedIn := c.Locals(constants.LoggedInSessionKey).(bool)
	if loggedIn {
		fiberlog.Error("logged in, redirecting to internal")
		return c.Redirect("/app/lists", fiber.StatusFound)
	}

	return c.Next()
}
