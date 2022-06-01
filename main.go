package main

import (
	"database/sql"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"hxann.com/blog/blog"
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

	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		sugar.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		sugar.Fatal(err)
	}
	sugar.Info("Database connected.")

	r := blog.NewRouter(sugar, db)

	sugar.Info("Server started on port " + port)

	http.ListenAndServe(":"+port, r)
}
