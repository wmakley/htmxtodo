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
	"htmxtodo/internal/view"
	"log"
	"net/http"
	"os"
	"strings"

	. "github.com/go-jet/jet/v2/postgres"
	"github.com/labstack/echo/v4"
	. "htmxtodo/gen/htmxtodo_dev/public/table"
)

//go:embed views
var viewsFS embed.FS

//go:embed public
var publicFS embed.FS

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.HTTPErrorHandler = customHTTPErrorHandler
	e.Renderer = view.NewCompiledOnDemandRenderer()

	//e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
	//	c.Logger().Debugf("Request Body: %s", reqBody)
	//	//c.Logger().Debugf("Response Body: %s", resBody)
	//}))

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/lists")
	})
	e.GET("/lists", NewListsIndexHandler(db))
	e.GET("/lists/:id", NewShowListHandler(db))
	e.POST("/lists", NewCreateListHandler(db))

	e.Logger.SetLevel(log2.DEBUG)

	e.Logger.Fatal(e.Start(":1323"))
}

func customHTTPErrorHandler(err error, c echo.Context) {
	// Always log:
	c.Logger().Error(err)

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	//err2 := c.NoContent(code)
	//if err2 != nil {
	//	panic(err2)
	//}
	errorPage := fmt.Sprintf("public/%d.html", code)
	blob, err2 := publicFS.ReadFile(errorPage)
	if err2 != nil {
		panic(err2)
	}

	if err2 = c.Blob(code, "text/html", blob); err2 != nil {
		panic(err2)
	}
}

func NewListsIndexHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {

		stmt := Lists.SELECT(Lists.AllColumns).ORDER_BY(Lists.Name.ASC())

		var results []model.Lists
		if err := stmt.QueryContext(c.Request().Context(), db, &results); err != nil {
			return err
		}

		if results == nil {
			results = make([]model.Lists, 0)
		}

		return c.Render(http.StatusOK, "lists/index", echo.Map{
			"Title": "Lists",
			"Lists": results,
		})
	}
}

type ShowListParams struct {
	ID int64 `param:"id"`
}

func NewShowListHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var params ShowListParams
		if err := c.Bind(&params); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		stmt := Lists.SELECT(Lists.AllColumns).WHERE(Lists.ID.EQ(Int(params.ID)))

		var result model.Lists
		if err := stmt.QueryContext(c.Request().Context(), db, &result); err != nil {
			return err
		}

		if result.ID == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "list not found")
		}

		return c.Render(http.StatusOK, "lists/show", echo.Map{
			"Title": "List",
			"List":  result,
		})
	}
}

type CreateListRequest struct {
	Name string `json:"name" form:"name" query:"name"`
}

func NewCreateListHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		json := CreateListRequest{}

		err := c.Bind(&json)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		//c.Logger().Debugf("Bound: %+v", json)

		json.Name = strings.TrimSpace(json.Name)
		if json.Name == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "name is required")
		}

		tx, err := db.BeginTx(c.Request().Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt := Lists.INSERT(Lists.Name).
			VALUES(json.Name).
			RETURNING(Lists.AllColumns)

		c.Logger().Debug(stmt.Sql())

		result := model.Lists{}
		err = stmt.QueryContext(c.Request().Context(), tx, &result)
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}

		return c.Redirect(http.StatusFound, fmt.Sprintf("/lists/%d", result.ID))
	}
}
