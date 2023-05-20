package users

import (
	"errors"
	"net/http"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/repository"
	"github.com/labstack/echo/v4"
)

// RegisterUser godoc
//
//	@Tags			user
//	@Summary		Регистрирует нового пользователя.
//	@Description	Регистрирует нового пользователя.
//	@ID				registerUser
//	@Accept			json
//	@Param			request	body	users.NewUserRequest	true	"New user request"
//	@Success		200
//	@Failure		409	{string}	string	"User already exists"
//	@Failure		500	{string}	string	"Internal Server Error"
//	@Router			/public/users/register [post]
func RegisterUser(service UserService) func(echo.Context) error {
	return func(c echo.Context) error {
		registerRequest := NewUserRequest{}
		if err := c.Bind(&registerRequest); err != nil {
			c.Logger().Error(err)
			return echo.ErrBadRequest
		}

		if err := c.Validate(registerRequest); err != nil {
			c.Logger().Error(err)
			return echo.ErrBadRequest
		}

		err := service.RegisterUser(c.Request().Context(), registerRequest)
		if err != nil {
			if errors.Is(err, repository.ErrAlreadyExists) {
				c.Logger().Errorf("handler: user already exists, %s", err)
				return echo.NewHTTPError(http.StatusConflict, "User already exists")
			}
			c.Logger().Error(err)
			return echo.ErrInternalServerError
		}

		return c.String(http.StatusCreated, "OK")
	}
}

// AuthUser godoc
//
//	@Tags			user
//	@Summary		Аутентифицирует существующего пользователя.
//	@Description	Аутентифицирует пользователя по паре логин/пароль и возвращает jwt токен.
//	@ID				authUser
//	@Accept			json
//	@Param			request	body	users.UserAuthRequest	true	"User auth request"
//	@Success		200
//	@Failure		404	{string}	string	"User not found"
//	@Failure		500	{string}	string	"Internal Server Error"
//	@Router			/public/users/auth [post]
func AuthUser(service UserService, secret string) func(ctx echo.Context) error {
	return func(c echo.Context) error {
		authRequest := UserAuthRequest{}
		if err := c.Bind(&authRequest); err != nil {
			c.Logger().Error(err)
			return echo.ErrBadRequest
		}

		if err := c.Validate(authRequest); err != nil {
			c.Logger().Error(err)
			return echo.ErrBadRequest
		}

		token, err := service.AuthUser(c.Request().Context(), authRequest, secret)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.Logger().Errorf("handler: user not found, %s", err)
				return echo.NewHTTPError(http.StatusNotFound, "User not found")
			}
			c.Logger().Error(err)
			return echo.ErrInternalServerError
		}

		return c.JSON(http.StatusOK, token)
	}
}
