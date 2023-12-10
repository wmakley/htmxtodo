package view

import (
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

func RenderComponent(c *fiber.Ctx, status int, component templ.Component) error {
	c.Status(status).Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(c.Context(), c)
}
