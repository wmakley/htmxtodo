package app

import (
	"database/sql"
	"errors"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	"htmxtodo/internal/repo"
	"htmxtodo/internal/view"
	errorviews "htmxtodo/views/errors"
	listviews "htmxtodo/views/lists"
	"net/http"
	"os"
	"strings"
)

// Config is the global config for the app router. Host and Port are needed for absolute URL generation.
type Config struct {
	Env  string
	Host string
	Port string
	Repo repo.Repository
}

func NewConfigFromEnvironment(repo repo.Repository) Config {
	return Config{
		Env:  os.Getenv("ENV"),
		Host: os.Getenv("HOST"),
		Port: os.Getenv("PORT"),
		Repo: repo,
	}
}

const Development = "development"
const Production = "production"
const csrfToken = "csrfToken"

func New(config *Config) *fiber.App {
	fiberlog.Debug("Starting app with config:", config)

	app := fiber.New(fiber.Config{
		AppName:      "HtmxTodo 0.1.0",
		ErrorHandler: errorHandler,
	})

	app.Use(logger.New(logger.Config{
		DisableColors: config.Env == Production,
	}))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: config.Env == Development,
	}))
	app.Use(compress.New())
	app.Use(helmet.New())
	app.Use(favicon.New())
	app.Use(csrf.New(csrf.Config{
		CookieName: "csrf_htmxtodo",
		ContextKey: csrfToken,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/lists", http.StatusFound)
	})

	lists := ListsHandlers{repo: config.Repo}

	app.Get("/lists", lists.Index)
	app.Get("/lists/:id", lists.Show)
	app.Post("/lists", lists.Create)
	app.Get("/lists/:id/edit", lists.Edit)
	app.Patch("/lists/:id", lists.Update)
	app.Delete("/lists/:id", lists.Delete)

	return app
}

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

	// Parameter decoding errors are errors bad route, meaning not found (but may also be bugs)
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

type ListsHandlers struct {
	repo repo.Repository
}

func (l *ListsHandlers) Index(c *fiber.Ctx) error {
	results, err := l.repo.FilterLists(c.Context())
	if err != nil {
		return err
	}

	viewObjects := make([]view.Card, len(results))
	for i, result := range results {
		viewObjects[i] = view.Card{
			EditingName: false,
			List:        result,
		}
	}

	return view.RenderComponent(c, 200, listviews.Index(viewObjects, model.List{}))
}

func (l *ListsHandlers) Show(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	_, err := l.repo.GetListById(c.Context(), params.ID)
	if err != nil {
		return err
	}

	panic("not implemented")
}

func (l *ListsHandlers) Edit(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	result, err := l.repo.GetListById(c.Context(), params.ID)
	if err != nil {
		return err
	}

	return view.RenderComponent(c, 200, listviews.Card(view.Card{
		EditingName: true,
		List:        result,
	}))
}

type CreateListRequest struct {
	Name string `json:"name" form:"name"`
}

func (l *ListsHandlers) Create(c *fiber.Ctx) error {
	var req CreateListRequest

	err := c.BodyParser(&req)
	if err != nil {
		return err
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fiber.NewError(http.StatusBadRequest, "name is required")
	}

	result, err := l.repo.CreateList(c.Context(), req.Name)
	if err != nil {
		return err
	}

	return view.RenderComponent(c, 200, listviews.Card(view.Card{
		EditingName: false,
		List:        result,
	}))
}

type UpdateListRequest struct {
	Name string `json:"name" form:"name"`
}

func (l *ListsHandlers) Update(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return err
	}

	var req UpdateListRequest
	if err := c.BodyParser(&req); err != nil {
		return err
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return fiber.NewError(http.StatusBadRequest, "name is required")
	}

	list, err := l.repo.UpdateListById(c.Context(), params.ID, req.Name)
	if err != nil {
		return err
	}

	return view.RenderComponent(c, 200, listviews.Card(view.Card{
		EditingName: false,
		List:        list,
	}))
}

func (l *ListsHandlers) Delete(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	err := l.repo.DeleteListById(c.Context(), params.ID)
	if err != nil {
		return err
	}

	return c.SendStatus(http.StatusNoContent)
}
