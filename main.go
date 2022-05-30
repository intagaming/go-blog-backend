package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"hxann.com/blog/blog"
)

func main() {
	log.Printf("Initializing...")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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

	log.Print("Server started")

	http.ListenAndServe(":8080", r)
}
