package clients

import (
	"context"
	"errors"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/auth"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/configs"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/repository"
	"github.com/c0dered273/automation-remote-controller/internal/user-account/users"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// ClientService сервис обрабатывает запросы с сущностями клиентских приложений
type ClientService interface {
	// NewClient сохраняет данные нового клиентского приложения и генерирует сертификат для идентификации клиента
	NewClient(ctx context.Context, clientName string, username string, caKeyPair auth.CertKeyPair) (auth.ClientCert, error)
}

type ClientServiceImpl struct {
	clientRepo   ClientRepository
	userRepo     users.UserRepository
	clientConfig configs.ClientConfig
}

func (c ClientServiceImpl) NewClient(
	ctx context.Context,
	clientName string,
	username string,
	caKeyPair auth.CertKeyPair,
) (auth.ClientCert, error) {
	clientID := uuid.NewString()

	tgName, err := c.userRepo.FindTGNameByUsername(ctx, username)
	if err != nil {
		return auth.ClientCert{}, err
	}

	newClient := Client{
		Name:       clientName,
		ClientUUID: clientID,
		OwnerName:  username,
	}
	err = c.clientRepo.SaveClient(ctx, newClient)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return auth.ClientCert{}, repository.ErrAlreadyExists
		}
		return auth.ClientCert{}, err
	}

	cert, err := auth.GenerateCert(
		caKeyPair,
		tgName,
		clientID,
		c.clientConfig.DomainName,
	)
	if err != nil {
		return auth.ClientCert{}, err
	}

	return auth.ClientCert{
		Cert: cert.Cert,
	}, nil
}

func NewClientService(client ClientRepository, userRepo users.UserRepository, clientConfig configs.ClientConfig) ClientServiceImpl {
	return ClientServiceImpl{
		clientRepo:   client,
		userRepo:     userRepo,
		clientConfig: clientConfig,
	}
}
