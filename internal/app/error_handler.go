package app

import (
	"database/sql"
	"errors"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"htmxtodo/internal/view"
	errorviews "htmxtodo/views/errors"
	"net/http"
	"strings"
)

func errorHandler(c *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := http.StatusInternalServerError
	msg := err.Error()

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Special handling for other error types
	if errors.Is(err, sql.ErrNoRows) {
		code = http.StatusNotFound
	}

	// Parameter decoding errors indicate user input did not match the route, i.e. not found (but may also be bugs)
	if strings.HasPrefix(msg, "failed to decode:") {
		code = http.StatusNotFound
	}

	// Render a template for 404 errors
	if code == http.StatusNotFound {
		return view.RenderComponent(c, code, errorviews.Error404())
	}

	// Log 500 errors and also render a default template
	fiberlog.Error(msg)
	return view.RenderComponent(c, code, errorviews.Error500())
}
