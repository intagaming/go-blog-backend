package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"hxann.com/blog/api/auth"
	"hxann.com/blog/api/resp"
	"hxann.com/blog/models"
	"hxann.com/blog/rate_limiting"
)

type RequestAuthorCtxKey struct{}
type PostCtxKey struct{}
type AuthorCtxKey struct{}

type Middleware struct {
	Sugar       *zap.SugaredLogger
	Authors     *models.AuthorModel
	Posts       *models.PostModel
	RedisClient *redis.Client
}

func (m *Middleware) AuthorizedRateLimiter(h http.Handler) http.Handler {
	return rate_limiting.NewHTTPRateLimiterHandler(h, m.Sugar, &rate_limiting.RateLimiterConfig{
		Extractor:   rate_limiting.NewHTTPHeadersExtractor("Authorization"),
		Strategy:    rate_limiting.NewSortedSetCounterStrategy(m.RedisClient, time.Now),
		Expiration:  10 * time.Second,
		MaxRequests: 30,
	})
}

var errInsufficientScope = errors.New("insufficient scope")

// RequiresAuthor requires the request to be authenticated as an author.
func (m *Middleware) RequiresAuthor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

		claims := token.CustomClaims.(*auth.CustomClaims)
		if !claims.HasScope("author") {
			render.Render(w, r, resp.ErrForbidden(errInsufficientScope))
			return
		}

		// attempt to set Author to context
		sub := token.RegisteredClaims.Subject
		author, err := m.Authors.Get(sub)
		if err != nil {
			if err == sql.ErrNoRows {
				// register Author if they are not in the db
				newAuthor := models.Author{UserId: sub, FullName: claims.Name, Email: claims.Email}
				m.Authors.Add(&newAuthor)
				author = &newAuthor
			} else {
				render.Render(w, r, resp.ErrInternal(err))
				panic(err)
			}
		}
		// set Author to the context
		ctx := context.WithValue(r.Context(), RequestAuthorCtxKey{}, author)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequiresAuthorOfPost requires the request to be authenticated as the author
// of the subjected post.
func (m *Middleware) RequiresAuthorOfPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		post := r.Context().Value(PostCtxKey{}).(*models.Post)
		author := r.Context().Value(RequestAuthorCtxKey{}).(*models.Author)

		if !post.IsAuthor(author) && !auth.IsAdmin(r) {
			render.Render(w, r, resp.ErrForbidden(errors.New("you must be the among the authors of the post in order to access this resource")))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequiresAdmin requires the request to be authenticated as an admin.
func (m *Middleware) RequiresAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdmin(r) {
			render.Render(w, r, resp.ErrForbidden(errInsufficientScope))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) PostContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var post *models.Post
		var err error

		if slug := chi.URLParam(r, "slug"); slug != "" {
			post, err = m.Posts.Get(slug)
		} else { // slug empty
			render.Render(w, r, resp.ErrInvalidRequest(errors.New("slug required")))
		}
		if err == sql.ErrNoRows {
			render.Render(w, r, resp.ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, resp.ErrInternal(err))
			panic(err)
		}
		ctx := context.WithValue(r.Context(), PostCtxKey{}, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) AuthorContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var author *models.Author
		var err error

		if userId := chi.URLParam(r, "user_id"); userId != "" {
			author, err = m.Authors.Get(userId)
		} else {
			render.Render(w, r, resp.ErrInvalidRequest(errors.New("user_id required")))
		}
		if err == sql.ErrNoRows {
			render.Render(w, r, resp.ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, resp.ErrInternal(err))
			panic(err)
		}
		ctx := context.WithValue(r.Context(), AuthorCtxKey{}, author)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
