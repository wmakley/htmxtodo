package main

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4/middleware"
	log2 "github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	repo2 "htmxtodo/internal/repo"
	"htmxtodo/internal/view"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

//go:embed all:views/*
var viewsFS embed.FS

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	env := os.Getenv("ENVIRONMENT")

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	repo := repo2.New(db)

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.HTTPErrorHandler = customHTTPErrorHandler

	e.Renderer = view.New(&view.Config{
		CompileOnRender: env == "development",
		Path:            "views",
		EmbedFS:         viewsFS,
	})

	//e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
	//	c.Logger().Debugf("Request Body: %s", reqBody)
	//	//c.Logger().Debugf("Response Body: %s", resBody)
	//}))

	r := e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/lists")
	})
	r.Name = "root"

	lists := ListsHandlers{repo: repo}

	r = e.GET("/lists", lists.Index)
	r.Name = "lists-index"
	r = e.GET("/lists/:id", lists.Show)
	r.Name = "lists-show"
	r = e.POST("/lists", lists.Create)
	r.Name = "lists-create"
	r = e.PATCH("/lists/:id", lists.Update)
	r.Name = "lists-update"

	e.Logger.SetLevel(log2.DEBUG)

	e.Logger.Fatal(e.Start(":1323"))
}

func customHTTPErrorHandler(err error, c echo.Context) {
	c.Logger().Error(err)

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	errorPage := fmt.Sprintf("errors/%d.html", code)

	if err2 := c.Render(code, errorPage, echo.Map{}); err2 != nil {
		panic(err2)
	}
}

type ListsHandlers struct {
	repo repo2.Repository
}

func (l *ListsHandlers) Index(c echo.Context) error {

	results, err := l.repo.FilterLists(c.Request().Context())
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "lists/index.html", echo.Map{
		"Title":   "Lists",
		"Lists":   results,
		"NewList": model.List{},
	})
}

type ShowListParams struct {
	ID int64 `param:"id"`
}

func (l *ListsHandlers) Show(c echo.Context) error {
	var params ShowListParams
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := l.repo.GetListById(c.Request().Context(), params.ID)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "lists/show.html", echo.Map{
		"Title": "List",
		"List":  result,
	})
}

func (l *ListsHandlers) Edit(c echo.Context) error {
	var params ShowListParams
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := l.repo.GetListById(c.Request().Context(), params.ID)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "lists/edit.html", echo.Map{
		"List": result,
	})
}

type CreateListRequest struct {
	Name string `json:"name" form:"name"`
}

func (l *ListsHandlers) Create(c echo.Context) error {
	var req CreateListRequest

	err := c.Bind(&req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	result, err := l.repo.CreateList(c.Request().Context(), req.Name)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "lists/shared/_card.html", result)
}

type UpdateListRequest struct {
	ID   int64  `param:"id"`
	Name string `json:"name" form:"name"`
}

func (l *ListsHandlers) Update(c echo.Context) error {
	var req UpdateListRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	list, err := l.repo.UpdateListById(c.Request().Context(), req.ID, req.Name)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, fmt.Sprintf("/lists/%d", list.ID))
}
