package clients

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/auth"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var (
	certFileName = "clientCert.pem"
)

// RegisterNewClient godoc
//
//	@Tags			client
//	@Summary		Регистрирует клиентское приложение для указанного пользователя.
//	@Description	Регистрирует клиентское приложение для указанного пользователя и возвращает pem файл с сертификатом/приватным ключом.
//	@ID				newClient
//	@Accept			json
//	@Param			client_name	path	string	true	"Register new client app"
//	@Success		200
//	@Failure		404	{string}	string	"User not found"
//	@Failure		500	{string}	string	"Internal Server Error"
//	@Router			/clients/{client_name}/register [put]
func RegisterNewClient(service ClientService, caKeyPair auth.CertKeyPair) func(ctx echo.Context) error {
	return func(c echo.Context) error {
		user := c.Get("user").(*jwt.Token)
		claims := user.Claims.(*auth.JwtCustomClaims)
		username := claims.Username
		clientName := c.Param("clientName")

		cert, err := service.NewClient(c.Request().Context(), clientName, username, caKeyPair)
		if err != nil {
			if errors.Is(err, repository.ErrAlreadyExists) {
				c.Logger().Errorf("handler: client already exists, %s", err)
				return echo.NewHTTPError(http.StatusConflict, "Client already exists")
			}
			c.Logger().Error(err)
			return echo.ErrInternalServerError
		}

		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", certFileName))
		return c.Blob(http.StatusOK, "application/octet-stream", cert.Cert)
	}
}
