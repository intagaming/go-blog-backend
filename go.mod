module hxann.com/blog

// +heroku goVersion go1.18
go 1.18

require github.com/auth0/go-jwt-middleware/v2 v2.0.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)

require (
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/cors v1.2.1
	github.com/go-chi/render v1.0.1
	github.com/go-sql-driver/mysql v1.6.0
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
)
