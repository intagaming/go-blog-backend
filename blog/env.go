package blog

import (
	"database/sql"

	"go.uber.org/zap"
	"hxann.com/blog/models"
)

type Env struct {
	sugar   *zap.SugaredLogger
	posts   models.PostModel
	authors models.AuthorModel
}

func GenerateEnv(sugar *zap.SugaredLogger, db *sql.DB) *Env {
	return &Env{
		sugar:   sugar,
		posts:   models.PostModel{DB: db},
		authors: models.AuthorModel{DB: db},
	}
}
