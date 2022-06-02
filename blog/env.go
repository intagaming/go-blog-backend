package blog

import (
	"database/sql"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"hxann.com/blog/models"
)

type Env struct {
	sugar       *zap.SugaredLogger
	posts       models.PostModel
	authors     models.AuthorModel
	redisClient *redis.Client
}

func GenerateEnv(sugar *zap.SugaredLogger, db *sql.DB, redisClient *redis.Client) *Env {
	return &Env{
		sugar:       sugar,
		posts:       models.PostModel{DB: db},
		authors:     models.AuthorModel{DB: db},
		redisClient: redisClient,
	}
}
