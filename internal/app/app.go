package app

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	_ "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/gofiber/fiber/v2"
	fiberlog "github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/postgres/v3"
	_ "github.com/lib/pq"
	"htmxtodo/gen/htmxtodo_dev/public/model"
	"htmxtodo/internal/config"
	"htmxtodo/internal/constants"
	"htmxtodo/internal/repo"
	"htmxtodo/internal/view"
	errorviews "htmxtodo/views/errors"
	listviews "htmxtodo/views/lists"
	loginviews "htmxtodo/views/login"
	"strings"
	"time"
)

func New(cfg *config.Config) *fiber.App {
	fiberlog.Info("Starting app with environment: ", cfg.Env)

	app := fiber.New(fiber.Config{
		AppName:      "HtmxTodo 0.1.0",
		ErrorHandler: errorHandler,
	})

	postgresStorage := postgres.New(postgres.Config{
		ConnectionURI: cfg.Secrets.DatabaseUrl(),
	})

	sessionStore := session.New(session.Config{
		Expiration:     24 * time.Hour * 30,
		KeyLookup:      "cookie:htmxtodo_session_id",
		CookieSecure:   cfg.CookieSecure,
		CookieHTTPOnly: true,
		Storage:        postgresStorage,
	})

	renderer := &view.Renderer{SessionStore: sessionStore}

	app.Use(logger.New(logger.Config{
		DisableColors: cfg.DisableLogColors,
	}))
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.EnableStackTrace,
	}))
	app.Use(compress.New())
	app.Use(helmet.New())
	app.Use(favicon.New())
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:       cfg.StaticFS,
		PathPrefix: "static",
	}))

	// Combine two CSRF extractors: use form field as default
	// so forms work without JS, with header as fallback.
	csrfFromForm := csrf.CsrfFromForm(constants.CsrfInputName)
	csrfFromHeader := csrf.CsrfFromHeader("X-CSRF-Token")

	app.Use(csrf.New(csrf.Config{
		CookieSecure: cfg.CookieSecure,
		Session:      sessionStore,
		Extractor: func(c *fiber.Ctx) (string, error) {
			token, err := csrfFromForm(c)
			if err == nil {
				return token, nil
			}

			if errors.Is(err, csrf.ErrMissingForm) {
				return csrfFromHeader(c)
			}

			// unexpected programmer error
			panic(err)
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			fiberlog.Error("CSRF error: ", err.Error())
			return renderer.RenderComponent(c, fiber.StatusForbidden,
				errorviews.GenericError(fiber.StatusForbidden, "Forbidden"))
		},
		ContextKey: constants.CsrfTokenContextKey,
		CookieName: "htmxtodo_csrf",
	}))

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		fiberlog.Fatal(err)
	}

	cognitoClient := cognito.NewFromConfig(awsCfg)

	login := LoginHandlers{
		renderer:        renderer,
		sessionStore:    sessionStore,
		cognitoClient:   cognitoClient,
		cognitoClientId: cfg.Secrets.CognitoClientId(),
	}

	lists := ListsHandlers{
		renderer:     renderer,
		repo:         cfg.Repo,
		sessionStore: sessionStore,
	}

	// check logged-in status on all routes
	app.Use(SetLoggedIn(sessionStore))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login", fiber.StatusFound)
	})

	internal := app.Group("/app", RequireLoggedIn)

	internal.Get("/lists", lists.Index)
	internal.Post("/lists", lists.Create)
	internal.Get("/lists/:id/edit", lists.Edit)
	internal.Patch("/lists/:id", lists.Update)
	internal.Delete("/lists/:id", lists.Delete)
	internal.Post("/logout", login.Logout)

	external := app.Group("", RedirectInternalIfLoggedIn)

	external.Get("/login", login.LoginForm)
	external.Post("/login", login.SubmitLogin)

	external.Get("/register", login.Register)
	external.Post("/register", login.SubmitRegistration)

	return app
}

type LoginHandlers struct {
	renderer        *view.Renderer
	sessionStore    *session.Store
	cognitoClient   *cognitoidentityprovider.Client
	cognitoClientId string
}

func (l *LoginHandlers) LoginForm(c *fiber.Ctx) error {
	form := loginviews.LoginForm{}
	return l.renderer.RenderComponent(c, 200, loginviews.Login(form))
}

func (l *LoginHandlers) SubmitLogin(c *fiber.Ctx) error {
	var form loginviews.LoginForm

	err := c.BodyParser(&form)
	if err != nil {
		return err
	}

	fiberlog.Debug("form:", form)

	sess, err := l.sessionStore.Get(c)
	if err != nil {
		panic(err)
	}

	if err = sess.Reset(); err != nil {
		panic(err)
	}
	sess.Set(constants.LoggedInSessionKey, "true")
	err = sess.Save()
	if err != nil {
		panic(err)
	}

	c.Set("HX-Location", "/app/lists")
	return c.Redirect("/app/lists", fiber.StatusFound)
}

func (l *LoginHandlers) Logout(c *fiber.Ctx) error {
	sess, err := l.sessionStore.Get(c)
	if err != nil {
		panic(err)
	}
	err = sess.Reset()
	if err != nil {
		panic(err)
	}

	c.Set("HX-Location", "/login")
	return c.Redirect("/login", fiber.StatusFound)
}

func (l *LoginHandlers) Register(c *fiber.Ctx) error {
	form := loginviews.RegistrationForm{}
	return l.renderer.RenderComponent(c, 200, loginviews.Register(form, ""))
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
		return l.renderer.RenderComponent(c, fiber.StatusUnprocessableEntity,
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
		return l.renderer.RenderComponent(c, fiber.StatusUnprocessableEntity, loginviews.Register(form, err.Error()))
	}

	c.Set("HX-Location", "/login")
	return c.Redirect("/login", fiber.StatusFound)
}

type ListsHandlers struct {
	renderer     *view.Renderer
	repo         repo.Repository
	sessionStore *session.Store
}

func (l *ListsHandlers) Index(c *fiber.Ctx) error {
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

	return l.renderer.RenderComponent(c, 200, listviews.Index(cards, newList))
}

func (l *ListsHandlers) Edit(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	result, err := l.repo.GetListById(c.Context(), params.ID)
	if err != nil {
		return err
	}

	return l.renderer.RenderComponent(c, 200, listviews.Card(listviews.CardProps{
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
		return l.renderer.RenderComponent(c, fiber.StatusUnprocessableEntity, listviews.CreateFailure(model.List{
			Name: req.Name,
		}, "name is required"))
	}

	result, err := l.repo.CreateList(c.Context(), req.Name)
	if err != nil {
		return err
	}

	return l.renderer.RenderComponent(c, 200, listviews.CreateSuccess(listviews.CardProps{
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
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	list, err := l.repo.UpdateListById(c.Context(), params.ID, req.Name)
	if err != nil {
		return err
	}

	return l.renderer.RenderComponent(c, 200, listviews.Card(listviews.CardProps{
		EditingName: false,
		List:        list,
	}))
}

func (l *ListsHandlers) Delete(c *fiber.Ctx) error {
	var params struct {
		ID int64 `params:"id"`
	}
	if err := c.ParamsParser(&params); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	err := l.repo.DeleteListById(c.Context(), params.ID)
	if err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
