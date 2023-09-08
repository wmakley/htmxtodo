package main

import (
	"database/sql"
	"embed"
	"errors"
	"github.com/gofiber/fiber/v2"
	globalLog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	repo2 "htmxtodo/internal/repo"
	"htmxtodo/internal/view"
	"log"
	"net/http"
	"os"
	"strings"
)

//go:embed all:views/*
var viewsFS embed.FS

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	env := os.Getenv("ENVIRONMENT")
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := repo2.New(db)

	app := fiber.New(fiber.Config{
		AppName:      "HtmxTodo 0.1.0",
		ErrorHandler: errorHandler,
		Views: view.New(&view.Config{
			CompileOnRender: env == "development",
			Path:            "views",
			EmbedFS:         viewsFS,
		}),
	})

	app.Use(logger.New(logger.Config{
		DisableColors: env != "development",
	}))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: env == "development",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/lists", http.StatusFound)
	})

	lists := ListsHandlers{repo: repo}

	app.Get("/lists", lists.Index)
	app.Get("/lists/:id", lists.Show)
	app.Post("/lists", lists.Create)
	app.Get("/lists/:id/edit", lists.Edit)
	app.Patch("/lists/:id", lists.Update)
	app.Delete("/lists/:id", lists.Delete)

	//e.Logger.SetLevel(log2.DEBUG)

	log.Fatal(app.Listen(host + ":" + port))
}

func errorHandler(c *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError
	msg := err.Error()

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Special handling for other error types
	if errors.Is(err, sql.ErrNoRows) {
		code = fiber.StatusNotFound
	}

	// Parameter decoding errors are errors bad route, meaning not found (but may also be bugs)
	if strings.HasPrefix(msg, "failed to decode:") {
		code = fiber.StatusNotFound
	}

	// Render a template for 404 errors
	if code == fiber.StatusNotFound {
		return c.Render("errors/404.html", fiber.Map{
			"Title":      "Error 404",
			"StatusCode": code,
		})
	}

	// Set Default Content-Type: text/plain; charset=utf-8
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)

	// Hide internal server error messages from external users
	if code == fiber.StatusInternalServerError {
		globalLog.Error(msg)
		msg = "500 Internal Server Error"
	}

	// Return status code with error message
	return c.Status(code).SendString(msg)
}

type ListsHandlers struct {
	repo repo2.Repository
}

type CardView struct {
	EditingName bool
	List        model.List
}

func (l *ListsHandlers) Index(c *fiber.Ctx) error {
	results, err := l.repo.FilterLists(c.Context())
	if err != nil {
		return err
	}

	viewObjects := make([]CardView, len(results))
	for i, result := range results {
		viewObjects[i] = CardView{
			EditingName: false,
			List:        result,
		}
	}

	return c.Render("lists/index.html", fiber.Map{
		"Title":   "Lists",
		"Lists":   viewObjects,
		"NewList": model.List{},
	})
}

func (l *ListsHandlers) Show(c *fiber.Ctx) error {
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

	return c.Render("lists/show.html", fiber.Map{
		"Title": "List",
		"List":  result,
	})
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

	return c.Render("lists/_card.html", CardView{
		EditingName: true,
		List:        result,
	})
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

	return c.Render("lists/_card.html", CardView{
		EditingName: false,
		List:        result,
	})
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

	return c.Render("lists/_card.html", CardView{
		EditingName: false,
		List:        list,
	})
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
