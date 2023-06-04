package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/c0dered273/automation-remote-controller/internal/tg-bot/repository"
	"github.com/jmoiron/sqlx"
)

type UserRepository interface {
	// UpdateChatIDByTGUser обновляет идентификатор чата, с которым работает пользователь
	UpdateChatIDByTGUser(ctx context.Context, tgName string, chatID int64) error
	// UpdateNotificationByTGUser изменение флага notification
	UpdateNotificationByTGUser(ctx context.Context, tgName string, isEnabled bool) error
	// FindUserByTGUser поиск пользователя по имени telegram
	FindUserByTGUser(ctx context.Context, tgUser string) (User, error)
	// IsUserExists проверяет наличие пользователя с указанным именем telegram
	IsUserExists(ctx context.Context, tgUser string) (bool, error)
	// FindUserByClientID поиск пользователя по идентификатору клиента
	FindUserByClientID(ctx context.Context, clientID string) (User, error)
}

type SQLUserRepo struct {
	db *sqlx.DB
}

func (r SQLUserRepo) UpdateChatIDByTGUser(ctx context.Context, tgName string, chatID int64) error {
	sqlQuery := `UPDATE users SET chat_id = $2 WHERE tg_user=$1`

	res, err := r.db.ExecContext(ctx, sqlQuery, tgName, chatID)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r SQLUserRepo) UpdateNotificationByTGUser(ctx context.Context, tgName string, isEnabled bool) error {
	sqlQuery := `UPDATE users SET notify_enabled = $2 WHERE tg_user=$1`

	res, err := r.db.ExecContext(ctx, sqlQuery, tgName, isEnabled)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r SQLUserRepo) FindUserByTGUser(ctx context.Context, tgUser string) (User, error) {
	sqlQuery := "SELECT username, tg_user, chat_id, notify_enabled FROM users WHERE tg_user = $1"

	user := User{}
	err := r.db.GetContext(ctx, &user, sqlQuery, tgUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, repository.ErrNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (r SQLUserRepo) IsUserExists(ctx context.Context, tgUser string) (bool, error) {
	sqlQuery := "SELECT 1 FROM users WHERE tg_user = $1"

	var result int
	err := r.db.GetContext(ctx, &result, sqlQuery, tgUser)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r SQLUserRepo) FindUserByClientID(ctx context.Context, clientID string) (User, error) {
	sqlQuery := "SELECT username, tg_user, chat_id, notify_enabled FROM users u JOIN clients c ON u.id = c.user_id WHERE c.uuid = $1"

	user := User{}
	err := r.db.GetContext(ctx, &user, sqlQuery, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, repository.ErrNotFound
		}
		return User{}, err
	}

	return user, nil
}

func NewRepo(db *sqlx.DB) SQLUserRepo {
	return SQLUserRepo{
		db: db,
	}
}
