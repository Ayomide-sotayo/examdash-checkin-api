package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	initDB()

	r := mux.NewRouter()

	// ---- Public routes — no token needed ----
	r.HandleFunc("/auth/signup", Signup).Methods("POST")
	r.HandleFunc("/auth/login",  Login).Methods("POST")

	// ---- Protected routes — JWT required ----
	// Learners see only their own checkins
	// Reviewers see everything
	r.HandleFunc("/checkins",      AuthMiddleware(GetCheckins)).Methods("GET")
	r.HandleFunc("/checkins",      AuthMiddleware(CreateCheckin)).Methods("POST")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(GetCheckinByID)).Methods("GET")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(PatchCheckin)).Methods("PATCH")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(DeleteCheckin)).Methods("DELETE")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server is running on :" + port + "...")
	http.ListenAndServe(":"+port, r)
}