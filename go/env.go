package openapi

import (
	"database/sql"

	"hxann.com/blog/models"
)

type Env struct {
	posts models.PostModel
}

func GenerateEnv(db *sql.DB) *Env {
	return &Env{
		posts: models.PostModel{DB: db},
	}
}
