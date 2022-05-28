package blog

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/go-chi/render"
	"hxann.com/blog/models"
)

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope string `json:"scope"`
	Email string `json:"https://hxann.com/email"`
	Name  string `json:"https://hxann.com/name"`
}

func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// EnsureValidToken is a middleware that will check the validity of our JWT.
func EnsureValidToken() func(next http.Handler) http.Handler {
	issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{os.Getenv("AUTH0_AUDIENCE")},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Encountered error while validating JWT: %v", err)

		render.Render(w, r, ErrUnauthorized(errors.New("failed to validate JWT")))
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return func(next http.Handler) http.Handler {
		return middleware.CheckJWT(next)
	}
}

// HasScope checks whether our claims have a specific scope.
func (c CustomClaims) HasScope(expectedScope string) bool {
	result := strings.Split(c.Scope, " ")
	for i := range result {
		if result[i] == expectedScope {
			return true
		}
	}

	return false
}

type requestAuthorCtxKey struct{}

var errInsufficientScope = errors.New("insufficient scope")

func (env *Env) AuthorEndpoint() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*CustomClaims)
			if !claims.HasScope("author") {
				render.Render(w, r, ErrForbidden(errInsufficientScope))
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
					render.Render(w, r, ErrInternal(err))
					panic(err)
				}
			}
			// set Author to the context
			ctx := context.WithValue(r.Context(), requestAuthorCtxKey{}, author)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (env *Env) AuthorOfPost() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			post := r.Context().Value(postCtxKey{}).(*models.Post)
			author := r.Context().Value(requestAuthorCtxKey{}).(*models.Author)

			if !post.IsAuthor(author) && !IsAdmin(r) {
				render.Render(w, r, ErrForbidden(errors.New("you must be the among the authors of the post in order to access this resource")))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (env *Env) AuthorOfPage() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page := r.Context().Value(pageCtxKey{}).(*models.Page)
			author := r.Context().Value(requestAuthorCtxKey{}).(*models.Author)

			if !page.IsAuthor(author) || !IsAdmin(r) {
				render.Render(w, r, ErrForbidden(errors.New("you must be the among the authors of the page in order to access this resource")))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func IsAdmin(r *http.Request) bool {
	token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	claims := token.CustomClaims.(*CustomClaims)
	return claims.HasScope("admin")
}

func (env *Env) AdminEndpoint(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r) {
			render.Render(w, r, ErrForbidden(errInsufficientScope))
			return
		}

		next.ServeHTTP(w, r)
	})
}
