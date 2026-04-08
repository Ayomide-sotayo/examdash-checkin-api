package main

import (
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Define Routes [cite: 7, 18]
	r.HandleFunc("/checkins", GetCheckins).Methods("GET")
	r.HandleFunc("/checkins", CreateCheckin).Methods("POST")
	r.HandleFunc("/checkins/{id}", PatchCheckin).Methods("PATCH")
	r.HandleFunc("/checkins/{id}", DeleteCheckin).Methods("DELETE")

	fmt.Println("Server is running on :8080...")
	http.ListenAndServe(":8080", r) // [cite: 18]
}