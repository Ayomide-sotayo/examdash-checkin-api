package main

import (
	"fmt"
	"net/http"
	"os"
	"github.com/gorilla/mux"
)

func main() {
	// Connect to Postgres first — crashes early if DB is unreachable
	initDB()

	r := mux.NewRouter()

	// All routes from Week 1 & 2 — kept exactly the same
	r.HandleFunc("/checkins",      GetCheckins).Methods("GET")
	r.HandleFunc("/checkins",      CreateCheckin).Methods("POST")
	r.HandleFunc("/checkins/{id}", GetCheckinByID).Methods("GET")
	r.HandleFunc("/checkins/{id}", PatchCheckin).Methods("PATCH")
	r.HandleFunc("/checkins/{id}", DeleteCheckin).Methods("DELETE")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server is running on :" + port + "...")
	http.ListenAndServe(":"+port, r)
}