package main

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"hxann.com/blog/api"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	sugar.Info("Initializing...")

	port := os.Getenv("PORT")
	if port == "" {
		sugar.Fatal("$PORT must be set")
	}

	// Initialize MySQL database
	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		sugar.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		sugar.Fatal(err)
	}
	sugar.Info("Database connected.")

	// Initialize Redis client
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		sugar.Fatal("$REDIS_URL must be set")
	}
	redisOpt, err := redis.ParseURL(redisUrl)
	if err != nil {
		sugar.Fatal("couldn't parse $REDIS_URL")
	}
	redisClient := redis.NewClient(redisOpt)

	// Create router
	r := api.NewRouter(sugar, db, redisClient)

	sugar.Info("Server started on port " + port)
	http.ListenAndServe(":"+port, r)
}
