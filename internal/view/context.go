package view

import "github.com/gofiber/fiber/v2"

// Context contains shared data needed by most templates, and wraps fiber.Ctx
type Context interface {
	Ctx() *fiber.Ctx
	CSRFToken() string
}

func NewContext(c *fiber.Ctx) Context {
	return &context{
		c: c,
	}
}

type context struct {
	c *fiber.Ctx
}

func (c *context) CSRFToken() string {
	return c.Ctx().Context().Value("csrfToken").(string)
}

func (c *context) Ctx() *fiber.Ctx {
	return c.c
}
