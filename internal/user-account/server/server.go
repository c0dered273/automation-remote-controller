package server

import (
	"os"
	"time"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/clients"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/configs"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/storage"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/users"
	"github.com/c0dered273/automation-remote-controller/pkg/auth"
	"github.com/c0dered273/automation-remote-controller/pkg/loggers"
	"github.com/c0dered273/automation-remote-controller/pkg/validators"
	"github.com/jmoiron/sqlx"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

var (
	LogWriter      = os.Stdout
	configFileName = "user_account_config"
	configFilePath = []string{
		".",
		"./configs/",
	}
)

type Services struct {
	UserService   users.UserService
	ClientService clients.ClientService
}

// NewServices настраивает сервисы
func NewServices(config *configs.UserAccountConfig, logger zerolog.Logger) (Services, *sqlx.DB) {
	// Users
	db, err := storage.NewConnection(config.DatabaseUri)
	if err != nil {
		logger.Fatal().Err(err).Msg("user_account: db connection error")
	}
	userRepo := users.NewRepo(db)
	userService := users.NewUserService(userRepo)

	// Clients
	clientRepo := clients.NewRepo(db)
	clientService := clients.NewClientService(clientRepo, userRepo, config.Client)

	return Services{
		UserService:   userService,
		ClientService: clientService,
	}, db
}

// ReadConfig читает и валидирует конфигурацию приложения
func ReadConfig() *configs.UserAccountConfig {
	logger := loggers.NewDefaultLogger(LogWriter)
	validator := validators.NewValidatorWithTagFieldName("mapstructure", logger)
	config, err := configs.NewUserAccountConfig(configFileName, configFilePath, logger, validator)
	if err != nil {
		logger.Fatal().Err(err).Msg("user_account: config init failed")
	}

	return config
}

// NewEchoServer возвращает настроенный сервер
func NewEchoServer(s Services, config *configs.UserAccountConfig, logger zerolog.Logger, validator validators.Validator) *echo.Echo {
	caKeyPair, err := auth.LoadKeyPair(config.CertFile, config.PKeyFile)
	if err != nil {
		logger.Fatal().Err(err).Msg("user_account: failed to load CA keys")
	}

	e := echo.New()
	e.Logger = loggers.NewEchoLogger(LogWriter, "echo", logger)
	e.Validator = validator
	e.Use(middleware.BodyLimit("10M"))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout:      15 * time.Second,
		ErrorMessage: "Connection timeout",
	}))
	e.Use(middleware.Decompress())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))
	e.Use(loggers.RequestLoggerMiddleware(logger))
	e.Use(middleware.Recover())

	// Public routes
	p := e.Group("/public")
	p.POST("/users/register", users.RegisterUser(s.UserService))
	p.POST("/users/auth", users.AuthUser(s.UserService, config.ApiSecret, config.JWTExpire))

	// Restricted routes
	r := e.Group("/")
	r.Use(echojwt.WithConfig(auth.GetJWTConfig(config.ApiSecret)))
	r.PUT("clients/:clientName/register", clients.RegisterNewClient(s.ClientService, caKeyPair))

	return e
}
