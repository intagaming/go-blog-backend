package blog

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

func NewRouter(db *sql.DB) *chi.Mux {
	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	env := GenerateEnv(db)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	r.Route("/posts", func(r chi.Router) {
		// Unauthenticated endpoints
		r.Get("/", env.PostsGet)

		// Authenticated endpoints for Authors
		r.Route("/", func(r chi.Router) {
			r.Use(EnsureValidToken())
			r.Use(env.AuthorEndpoint())

			r.Post("/", env.PostsPost)

			r.Route("/{slug}", func(r chi.Router) {
				r.Use(env.PostContext)
				r.Get("/", env.PostGet)
				r.Put("/", env.PostPut)
				r.Delete("/", env.PostDelete)
			})
		})
	})

	return r
}
