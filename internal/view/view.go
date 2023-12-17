package view

import (
	"context"
	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

type Renderer struct {
	SessionStore *session.Store
}

func (r *Renderer) Globals(c *fiber.Ctx) Globals {
	sess, err := r.SessionStore.Get(c)
	if err != nil {
		panic(err)
	}

	return &globals{
		ctx:  c,
		sess: sess,
	}
}

// Basic render, no globals in context
func RenderComponent(c *fiber.Ctx, status int, component templ.Component) error {
	c.Status(status).Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(c.Context(), c)
}

// Adds globals to context
func (r *Renderer) RenderComponent(c *fiber.Ctx, status int, component templ.Component) error {
	c.Status(status).Set("Content-Type", "text/html; charset=utf-8")
	ctx := context.WithValue(c.Context(), "globals", r.Globals(c))
	return component.Render(ctx, c)
}

type Globals interface {
	CSRFToken() string
	IsLoggedIn() bool
}

type globals struct {
	ctx  *fiber.Ctx
	sess *session.Session
}

func (g *globals) CSRFToken() string {
	panic("not implemented")
}

func (g *globals) IsLoggedIn() bool {
	return g.sess.Get("logged_in") == "true"
}
