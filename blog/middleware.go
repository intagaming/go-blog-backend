package blog

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-chi/render"
	"hxann.com/blog/blog/auth"
	"hxann.com/blog/blog/rate_limiting"
	"hxann.com/blog/blog/resp"
	"hxann.com/blog/models"
)

func (env *Env) AuthorizedRateLimiter(h http.Handler) http.Handler {
	return rate_limiting.NewHTTPRateLimiterHandler(h, env.sugar, &rate_limiting.RateLimiterConfig{
		Extractor:   rate_limiting.NewHTTPHeadersExtractor("Authorization"),
		Strategy:    rate_limiting.NewSortedSetCounterStrategy(env.redisClient, time.Now),
		Expiration:  10 * time.Second,
		MaxRequests: 30,
	})
}

type requestAuthorCtxKey struct{}

var errInsufficientScope = errors.New("insufficient scope")

// RequiresAuthor requires the request to be authenticated as an author.
func (env *Env) RequiresAuthor() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*auth.CustomClaims)
			if !claims.HasScope("author") {
				render.Render(w, r, resp.ErrForbidden(errInsufficientScope))
				return
			}

			// attempt to set Author to context
			sub := token.RegisteredClaims.Subject
			author, err := env.authors.Get(sub)
			if err != nil {
				if err == sql.ErrNoRows {
					// register Author if they are not in the db
					newAuthor := models.Author{UserId: sub, FullName: claims.Name, Email: claims.Email}
					env.authors.Add(&newAuthor)
					author = &newAuthor
				} else {
					render.Render(w, r, resp.ErrInternal(err))
					panic(err)
				}
			}
			// set Author to the context
			ctx := context.WithValue(r.Context(), requestAuthorCtxKey{}, author)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequiresAuthorOfPost requires the request to be authenticated as the author
// of the subjected post.
func (env *Env) RequiresAuthorOfPost() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			post := r.Context().Value(postCtxKey{}).(*models.Post)
			author := r.Context().Value(requestAuthorCtxKey{}).(*models.Author)

			if !post.IsAuthor(author) && !IsAdmin(r) {
				render.Render(w, r, resp.ErrForbidden(errors.New("you must be the among the authors of the post in order to access this resource")))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IsAdmin checks if the request is authenticated as an admin.
func IsAdmin(r *http.Request) bool {
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*auth.CustomClaims)
	return claims.HasScope("admin")
}

// RequiresAdmin requires the request to be authenticated as an admin.
func (env *Env) RequiresAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r) {
			render.Render(w, r, resp.ErrForbidden(errInsufficientScope))
			return
		}

		next.ServeHTTP(w, r)
	})
}
