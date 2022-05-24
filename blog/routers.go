package blog

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func NewRouter(db *sql.DB) *chi.Mux {
	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	env := GenerateEnv(db)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	r.Route("/posts", func(r chi.Router) {
		r.Get("/", env.PostsGet)
		r.Post("/", env.PostsPost)

		r.Route("/{slug}", func(r chi.Router) {
			r.Use(env.PostContext)
			r.Get("/", env.PostGet)
			r.Put("/", env.PostPut)
			r.Delete("/", env.PostDelete)
		})
	})

	return r
}
