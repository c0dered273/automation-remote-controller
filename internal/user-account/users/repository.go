package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/c0dered273/automation-remote-controller/internal/user-account/repository"
	"github.com/jmoiron/sqlx"
)

// UserRepository описывает методы работы с сущностью пользователя
type UserRepository interface {
	// SaveUser Сохранение нового пользователя
	SaveUser(ctx context.Context, user User) error
	// FindByNameAndPassword поиск пользователя по имени и паролю
	FindByNameAndPassword(ctx context.Context, name string, password string) (User, error)
	// FindTGNameByUsername поиск пользователя по нику telegram
	FindTGNameByUsername(ctx context.Context, name string) (string, error)
}

type SQLUserRepo struct {
	db *sqlx.DB
}

func (r SQLUserRepo) SaveUser(ctx context.Context, user User) error {
	const sqlQuery = `INSERT INTO users(username, password, tg_user, notify_enabled) 
			VALUES(:username, crypt(:password, gen_salt('bf')), :tg_user, true) 
			ON CONFLICT DO NOTHING`

	res, err := r.db.NamedExecContext(ctx, sqlQuery, user)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return repository.ErrAlreadyExists
	}

	return nil
}

func (r SQLUserRepo) FindByNameAndPassword(ctx context.Context, name string, password string) (User, error) {
	const sqlQuery = "SELECT username, tg_user FROM users WHERE username=$1 AND password=crypt($2, password)"

	user := User{}
	err := r.db.GetContext(ctx, &user, sqlQuery, name, password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, repository.ErrNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (r SQLUserRepo) FindTGNameByUsername(ctx context.Context, name string) (string, error) {
	const sqlQuery = `SELECT tg_user FROM users WHERE username=$1`

	var tgName string
	err := r.db.GetContext(ctx, &tgName, sqlQuery, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", repository.ErrNotFound
		}
		return "", err
	}

	return tgName, nil
}

func NewRepo(db *sqlx.DB) SQLUserRepo {
	return SQLUserRepo{
		db: db,
	}
}
