package storage

import (
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	MaxOpenConns    = 10
	MaxConnLifetime = time.Minute
)

// NewConnection возвращает настроенное соединение в БД
// используется драйвер pgx для PostgreSQL
func NewConnection(uri string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("pgx", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(MaxOpenConns)
	db.SetConnMaxLifetime(MaxConnLifetime)

	return db, nil
}
