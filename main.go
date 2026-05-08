package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "examdash-checkin-api/docs"
)

// @title           ExamDash Learner Check-in API
// @version         5.0
// @description     An authenticated REST API for managing learner check-ins at ExamDash. Supports JWT auth, ownership rules, and role-based access for learners and reviewers.

// @contact.name    Ayomide Sotayo
// @contact.url     https://github.com/Ayomide-sotayo/examdash-checkin-api

// @host            examdash-checkin-api-1.onrender.com
// @BasePath        /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token in the format: Bearer {token}

// corsMiddleware adds the headers needed for Swagger UI to call the API from the browser
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	initDB()

	r := mux.NewRouter()

	// ---- Swagger UI ----
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// ---- Public routes — no token needed ----
	r.HandleFunc("/auth/signup", Signup).Methods("POST", "OPTIONS")
	r.HandleFunc("/auth/login",  Login).Methods("POST", "OPTIONS")

	// ---- Protected routes — JWT required ----
	r.HandleFunc("/checkins",      AuthMiddleware(GetCheckins)).Methods("GET", "OPTIONS")
	r.HandleFunc("/checkins",      AuthMiddleware(CreateCheckin)).Methods("POST", "OPTIONS")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(GetCheckinByID)).Methods("GET", "OPTIONS")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(PatchCheckin)).Methods("PATCH", "OPTIONS")
	r.HandleFunc("/checkins/{id}", AuthMiddleware(DeleteCheckin)).Methods("DELETE", "OPTIONS")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server is running on :" + port + "...")
	fmt.Println("Swagger UI: https://examdash-checkin-api-1.onrender.com/swagger/index.html")

	// Wrap the entire router with CORS middleware
	http.ListenAndServe(":"+port, corsMiddleware(r))
}