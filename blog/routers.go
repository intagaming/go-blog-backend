package blog

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"hxann.com/blog/blog/auth"
	"hxann.com/blog/blog/logger"
)

func NewRouter(sugar *zap.SugaredLogger, db *sql.DB, redisClient *redis.Client) *chi.Mux {
	env := GenerateEnv(sugar, db, redisClient)
	httpLogger := &logger.HTTPLogger{
		Sugar: sugar,
	}
	ensureValidToken := auth.EnsureValidToken(sugar)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpLogger.LogRequestHandler)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	r.Route("/posts", func(r chi.Router) {
		// Unauthenticated endpoints
		r.Get("/", env.PostsGet)
		r.With(env.PostContext).Get("/{slug}", env.PostGet)

		// Authenticated endpoints for Authors
		r.Route("/", func(r chi.Router) {
			r.Use(ensureValidToken)
			r.Use(env.AuthorizedRateLimiter)
			r.Use(env.RequiresAuthor())

			r.Post("/", env.PostsPost)

			r.Route("/{slug}", func(r chi.Router) {
				r.Use(env.PostContext)
				r.Use(env.RequiresAuthorOfPost()) // requires author to be among the authors of the post
				r.Put("/", env.PostPut)
				r.Delete("/", env.PostDelete)
			})
		})
	})

	r.Route("/authors", func(r chi.Router) {
		r.Route("/me", func(r chi.Router) {
			r.Use(ensureValidToken)
			r.Use(env.AuthorizedRateLimiter)
			r.Use(env.RequiresAuthor())

			r.Get("/", env.AuthorsMeGet)
			r.Put("/", env.AuthorsMePut)
		})

		r.Route("/{user_id}", func(r chi.Router) {
			r.Use(env.AuthorContext)
			r.Get("/", env.AuthorGet)
			r.With(ensureValidToken, env.AuthorizedRateLimiter, env.RequiresAdmin).Put("/", env.AuthorPut)
		})
	})

	return r
}
