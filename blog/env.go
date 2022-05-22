package blog

import (
	"database/sql"

	"hxann.com/blog/models"
)

type Env struct {
	posts   models.PostModel
	authors models.AuthorModel
}

func GenerateEnv(db *sql.DB) *Env {
	return &Env{
		posts:   models.PostModel{DB: db},
		authors: models.AuthorModel{DB: db},
	}
}
