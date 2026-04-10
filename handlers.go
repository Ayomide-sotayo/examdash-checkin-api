package main

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
)

// GET /checkins - Returns all records
func GetCheckins(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkins)
}

// POST /checkins - Adds a new record
func CreateCheckin(w http.ResponseWriter, r *http.Request) {
	var newCheckin Checkin
	_ = json.NewDecoder(r.Body).Decode(&newCheckin) // Turn JSON into Go struct
	checkins = append(checkins, newCheckin)        // Add to our list
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newCheckin)
}

// PATCH /checkins/{id} - Updates a record
func PatchCheckin(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r) // Get the {id} from URL
	for index, item := range checkins {
		if item.ID == params["id"] {
			var updatedData Checkin
			_ = json.NewDecoder(r.Body).Decode(&updatedData)
			if updatedData.Status != "" {
				checkins[index].Status = updatedData.Status
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(checkins[index])
			return
		}
	}
}

// DELETE /checkins/{id} - Removes a record
func DeleteCheckin(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range checkins {
		if item.ID == params["id"] {
			// Remove the item by joining everything before it and everything after it
			checkins = append(checkins[:index], checkins[index+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
}