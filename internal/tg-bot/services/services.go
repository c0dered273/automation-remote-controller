package services

import (
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/storage"
	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/users"
	"github.com/rs/zerolog"
)

type Services struct {
	UserService users.UserService
}

func NewServices(databaseUri string, logger zerolog.Logger) Services {
	db, err := storage.NewConnection(databaseUri)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
	usersRepo := users.NewRepo(db)
	userService := users.NewUserService(usersRepo)

	return Services{
		UserService: userService,
	}
}
