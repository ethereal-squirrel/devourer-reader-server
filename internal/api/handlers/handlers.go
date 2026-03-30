package handlers

import (
	"database/sql"

	"github.com/devourer/server/internal/config"
)

type Watcher interface {
	Restart()
}

type Handlers struct {
	DB      *sql.DB
	Cfg     *config.Config
	Watcher Watcher
}

func New(d *sql.DB, cfg *config.Config) *Handlers {
	return &Handlers{DB: d, Cfg: cfg}
}
