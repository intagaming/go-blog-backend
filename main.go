package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

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
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		sugar.Fatal("$REDIS_ADDR must be set")
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	var redisDb int
	if redisDbStr := os.Getenv("REDIS_DB"); redisDbStr != "" {
		redisDbInt, err := strconv.Atoi(redisDbStr)
		if err != nil {
			sugar.Fatal("cannot parse $REDIS_DB to int")
		}
		redisDb = redisDbInt
	} else {
		redisDb = 0
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDb,
	})

	// Create router
	r := api.NewRouter(sugar, db, redisClient)

	sugar.Info("Server started on port " + port)
	http.ListenAndServe(":"+port, r)
}
