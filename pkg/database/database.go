package database

import (
	"net/url"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgreSQL driver
)

type Config struct {
	User     string
	Password string
	Host     string
	DbName   string
}

func Open(cfg Config) (*sqlx.DB, error) {

	q := make(url.Values)
	q.Set("sslmode", "disable")

	// URL example: postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.DbName,
		RawQuery: q.Encode(),
	}

	return sqlx.Open("postgres", u.String())
}
