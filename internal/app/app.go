package app

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	"htmxtodo/internal/repo"
	"htmxtodo/internal/view"
	listviews "htmxtodo/views/lists"
	loginviews "htmxtodo/views/login"
	"net/http"
	"os"
	"strings"

	_ "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
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
	//app.Use(csrf.New(csrf.Config{
	//	CookieName: "csrf_htmxtodo",
	//	ContextKey: csrfToken,
	//}))

	sessionStore := session.New()

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		fiberlog.Fatal(err)
	}

	cognitoClient := cognito.NewFromConfig(awsCfg)

	login := LoginHandlers{
		sessionStore:    sessionStore,
		cognitoClient:   cognitoClient,
		cognitoClientId: os.Getenv("COGNITO_CLIENT_ID"),
	}

	lists := ListsHandlers{
		repo:         config.Repo,
		sessionStore: sessionStore,
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", http.StatusFound)
	})
	app.Get("/login", login.LoginForm)
	app.Post("/login", login.SubmitLogin)

	app.Get("/register", login.Register)
	app.Post("/register", login.SubmitRegistration)

	internal := app.Group("")

	internal.Get("/lists", lists.Index)
	internal.Post("/lists", lists.Create)
	internal.Get("/lists/:id/edit", lists.Edit)
	internal.Patch("/lists/:id", lists.Update)
	internal.Delete("/lists/:id", lists.Delete)

	return app
}

type LoginHandlers struct {
	sessionStore    *session.Store
	cognitoClient   *cognitoidentityprovider.Client
	cognitoClientId string
}

func (l *LoginHandlers) LoginForm(c *fiber.Ctx) error {
	form := loginviews.LoginForm{}
	return view.RenderComponent(c, 200, loginviews.Login(form))
}

func (l *LoginHandlers) SubmitLogin(c *fiber.Ctx) error {
	var form loginviews.LoginForm

	err := c.BodyParser(&form)
	if err != nil {
		return err
	}

	fiberlog.Debug("form:", form)
	return view.RenderComponent(c, 200, loginviews.Login(form))
}

func (l *LoginHandlers) Register(c *fiber.Ctx) error {
	form := loginviews.RegistrationForm{}
	return view.RenderComponent(c, 200, loginviews.Register(form, ""))
}

func (l *LoginHandlers) SubmitRegistration(c *fiber.Ctx) error {
	var form loginviews.RegistrationForm

	err := c.BodyParser(&form)
	if err != nil {
		return err
	}

	fiberlog.Debug("form:", form)

	// validation:

	if form.Password != form.PasswordConfirmation {
		return view.RenderComponent(c, http.StatusUnprocessableEntity,
			loginviews.Register(form, "passwords do not match"))
	}

	signUpInput := &cognito.SignUpInput{
		ClientId:   aws.String(l.cognitoClientId),
		Password:   aws.String(form.Password),
		Username:   aws.String(form.Email),
		SecretHash: nil,
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("email"),
				Value: aws.String(form.Email),
			},
		},
		UserContextData: &types.UserContextDataType{
			EncodedData: nil,
			IpAddress:   aws.String(c.IP()),
		},
		ValidationData: nil,
	}

	_, err = l.cognitoClient.SignUp(c.Context(), signUpInput)
	if err != nil {
		return view.RenderComponent(c, http.StatusUnprocessableEntity, loginviews.Register(form, err.Error()))
	}

	return c.Redirect("/lists", http.StatusFound)
}

type ListsHandlers struct {
	repo         repo.Repository
	sessionStore *session.Store
	clerk        clerk.Client
}

func (l *ListsHandlers) Index(c *fiber.Ctx) error {
	sessionClaims, ok := clerk.SessionFromContext(c.Context())
	if ok {
		jsonResp, err := json.Marshal(sessionClaims)
		if err != nil {
			panic(err)
		}
		fiberlog.Debug(string(jsonResp))
	} else {
		fiberlog.Error("could not get clerk session")
	}

	results, err := l.repo.FilterLists(c.Context())
	if err != nil {
		return err
	}

	cards := make([]listviews.CardProps, len(results))
	for i, result := range results {
		cards[i] = listviews.CardProps{
			EditingName: false,
			List:        result,
		}
	}
	newList := model.List{}

	return view.RenderComponent(c, 200, listviews.Index(cards, newList))
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

	return view.RenderComponent(c, 200, listviews.Card(listviews.CardProps{
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
		return view.RenderComponent(c, http.StatusUnprocessableEntity, listviews.CreateFailure(model.List{
			Name: req.Name,
		}, "name is required"))
	}

	result, err := l.repo.CreateList(c.Context(), req.Name)
	if err != nil {
		return err
	}

	return view.RenderComponent(c, 200, listviews.CreateSuccess(listviews.CardProps{
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

	return view.RenderComponent(c, 200, listviews.Card(listviews.CardProps{
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
