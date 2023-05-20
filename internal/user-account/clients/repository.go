package clients

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// ClientRepository описывает методы работы с данными клиентских приложений
type ClientRepository interface {
	// SaveClient сохраняет сущность client
	SaveClient(ctx context.Context, client Client) error
}

// SQLClientRepo для хранения данных используется стандартный пакет database/sql c оберткой sqlx
type SQLClientRepo struct {
	db *sqlx.DB
}

func (r SQLClientRepo) SaveClient(ctx context.Context, client Client) error {
	sqlQuery := `INSERT INTO clients(name, uuid, user_id) 
					VALUES(:name, :uuid, (SELECT id FROM users u WHERE u.username=:username))`

	_, err := r.db.NamedExecContext(ctx, sqlQuery, client)
	if err != nil {
		return err
	}

	return nil
}

func NewClientRepo(db *sqlx.DB) SQLClientRepo {
	return SQLClientRepo{
		db: db,
	}
}
