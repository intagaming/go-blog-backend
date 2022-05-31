package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"hxann.com/blog/blog"
)

func main() {
	log.Printf("Initializing...")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Database connected.")

	r := blog.NewRouter(db)

	log.Print("Server started on port " + port)

	http.ListenAndServe(":"+port, r)
}
