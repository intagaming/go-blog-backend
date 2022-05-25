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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Failed to validate JWT."}`))
		// TODO: standardize error response body
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

type authorCtxKey struct{}

func (env *Env) AuthorEndpoint() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)

			claims := token.CustomClaims.(*CustomClaims)
			if !claims.HasScope("author") && !claims.HasScope("admin") {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"message":"Insufficient scope."}`))
				// TODO: standardize response
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
			ctx := context.WithValue(r.Context(), authorCtxKey{}, author)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (env *Env) AuthorOfPost() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			post := r.Context().Value(postCtxKey{}).(*models.Post)
			author := r.Context().Value(authorCtxKey{}).(*models.Author)

			if !post.IsAuthor(author) {
				render.Render(w, r, ErrForbidden(errors.New("you must be the among the authors of the post in order to access this resource")))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
