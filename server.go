package main

import (
	"database/sql"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4/middleware"
	log2 "github.com/labstack/gommon/log"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"

	. "htmxtodo/gen/htmxtodo_dev/public/table"
)

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
	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		c.Logger().Debugf("Request Body: %s", reqBody)
		//c.Logger().Debugf("Response Body: %s", resBody)
	}))

	e.GET("/lists", NewIndexHandler(db))
	e.POST("/lists", NewCreateListHandler(db))

	e.Logger.SetLevel(log2.DEBUG)

	e.Logger.Fatal(e.Start(":1323"))
}

func NewIndexHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		tx, err := db.BeginTx(c.Request().Context(), nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt := Lists.SELECT(Lists.AllColumns).ORDER_BY(Lists.Name.ASC())
		c.Logger().Debug(stmt.Sql())

		var results []model.Lists
		err = stmt.QueryContext(c.Request().Context(), tx, &results)
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		if results == nil {
			results = make([]model.Lists, 0)
		}

		err = tx.Commit()
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		return c.JSON(http.StatusOK, results)
	}
}

type CreateListRequest struct {
	Name string `json:"name" form:"name" query:"name"`
}

func NewCreateListHandler(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Header.Get("Content-Type") != "application/json" {
			return echo.NewHTTPError(http.StatusBadRequest, "Content-Type must be application/json")
		}

		json := CreateListRequest{}

		err := c.Bind(&json)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		c.Logger().Debugf("Bound: %+v", json)

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
			c.Logger().Error(err)
			return err
		}

		err = tx.Commit()
		if err != nil {
			c.Logger().Error(err)
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}
