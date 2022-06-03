package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"hxann.com/blog/api/auth"
	"hxann.com/blog/api/handlers"
	"hxann.com/blog/api/logger"
	blogMiddleware "hxann.com/blog/api/middleware"
	"hxann.com/blog/models"
)

func NewRouter(sugar *zap.SugaredLogger, db *sql.DB, redisClient *redis.Client) *chi.Mux {
	// Initialize some middleware
	httpLogger := &logger.HTTPLogger{
		Sugar: sugar,
	}
	ensureValidToken := auth.EnsureValidToken(sugar)

	postsModel := &models.PostModel{DB: db}
	authorsModel := &models.AuthorModel{DB: db}
	middleware := blogMiddleware.Middleware{
		Sugar:       sugar,
		Authors:     authorsModel,
		Posts:       postsModel,
		RedisClient: redisClient,
	}

	// Create new router
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpLogger.LogRequestHandler)
	r.Use(chiMiddleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	posts := handlers.NewPosts(postsModel, authorsModel)
	r.Route("/posts", func(r chi.Router) {
		// Unauthenticated endpoints
		r.Get("/", posts.PostsGet)
		r.With(middleware.PostContext).Get("/{slug}", posts.PostGet)

		// Authenticated endpoints for Authors
		r.Route("/", func(r chi.Router) {
			r.Use(ensureValidToken)
			r.Use(middleware.AuthorizedRateLimiter)
			r.Use(middleware.RequiresAuthor)

			r.Post("/", posts.PostsPost)

			r.Route("/{slug}", func(r chi.Router) {
				r.Use(middleware.PostContext)
				r.Use(middleware.RequiresAuthorOfPost) // requires author to be among the authors of the post
				r.Put("/", posts.PostPut)
				r.Delete("/", posts.PostDelete)
			})
		})
	})

	authors := handlers.NewAuthors(authorsModel)
	r.Route("/authors", func(r chi.Router) {
		r.Route("/me", func(r chi.Router) {
			r.Use(ensureValidToken)
			r.Use(middleware.AuthorizedRateLimiter)
			r.Use(middleware.RequiresAuthor)

			r.Get("/", authors.AuthorsMeGet)
			r.Put("/", authors.AuthorsMePut)
		})

		r.Route("/{user_id}", func(r chi.Router) {
			r.Use(middleware.AuthorContext)
			r.Get("/", authors.AuthorGet)
			r.With(
				ensureValidToken,
				middleware.AuthorizedRateLimiter,
				middleware.RequiresAdmin,
			).Put("/", authors.AuthorPut)
		})
	})

	return r
}
