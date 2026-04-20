package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	// Week 1 routes (kept exactly as before)
	r.HandleFunc("/checkins", GetCheckins).Methods("GET")
	r.HandleFunc("/checkins", CreateCheckin).Methods("POST")
	r.HandleFunc("/checkins/{id}", PatchCheckin).Methods("PATCH")
	r.HandleFunc("/checkins/{id}", DeleteCheckin).Methods("DELETE")

	// Week 2 new route
	r.HandleFunc("/checkins/{id}", GetCheckinByID).Methods("GET")

	port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}
fmt.Println("Server is running on :" + port + "...")
http.ListenAndServe(":"+port, r)
}