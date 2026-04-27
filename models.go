package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver — blank import registers it
)

// Checkin is the core domain struct.
// Week 3 adds CreatedAt and UpdatedAt (audit fields).
type Checkin struct {
	ID          string `json:"id"`
	LearnerName string `json:"learner_name"`
	Track       string `json:"track"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// DB is the global database connection pool.
// It's safe for concurrent use — database/sql manages pooling internally.
var DB *sql.DB

// initDB opens the connection and verifies it with a ping.
// Called once at startup from main().
func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Ping confirms the credentials and network are actually working
	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connected successfully")
}