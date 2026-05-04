package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// Checkin is the core domain struct.
// Week 4 adds UserID to link each checkin to the user who created it.
type Checkin struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id,omitempty"`
	LearnerName string `json:"learner_name"`
	Track       string `json:"track"`
	Status      string `json:"status"`
	SubmittedAt string `json:"submitted_at"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// User represents a registered user in the system.
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
}

// DB is the global database connection pool.
var DB *sql.DB

// initDB opens the connection and verifies it with a ping.
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

	if err = DB.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connected successfully")
}