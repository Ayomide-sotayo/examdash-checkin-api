package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"github.com/gorilla/mux"
)

// ---- Helper: write a JSON error response ----
// This is a small reusable function so we don't repeat ourselves.
// Every error in this API has the same shape: {"error": "...", "message": "..."}
func writeError(w http.ResponseWriter, statusCode int, errType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   errType,
		"message": message,
	})
}

// ---- Helper: validate a Checkin struct ----
// Returns an error message string, or "" if everything is fine.
// We call this inside POST and PATCH before saving anything.
func validateCheckin(c Checkin) string {
	// learner_name must not be empty or just spaces
	if strings.TrimSpace(c.LearnerName) == "" {
		return "learner_name is required and cannot be empty"
	}

	// track must be one of the allowed values
	validTracks := map[string]bool{
		"Backend": true, "Frontend": true,
		"Product Design": true, "Product Management": true, "Growth": true,
	}
	if !validTracks[c.Track] {
		return "track must be one of: Backend, Frontend, Product Design, Product Management, Growth"
	}

	// status must be one of the allowed values
	validStatuses := map[string]bool{
		"pending": true, "submitted": true, "reviewed": true,
	}
	if !validStatuses[c.Status] {
		return "status must be one of: pending, submitted, reviewed"
	}

	// submitted_at must not be empty
	if strings.TrimSpace(c.SubmittedAt) == "" {
		return "submitted_at is required"
	}

	return "" // empty string = no error
}

// ---- GET /checkins ----
// Returns all records. Also supports:
//   ?track=Backend       → filter by track
//   ?status=submitted    → filter by status
//   ?sort=submitted_at   → sort ascending by submitted_at
func GetCheckins(w http.ResponseWriter, r *http.Request) {
	// Read query parameters from the URL
	trackFilter := r.URL.Query().Get("track")
	statusFilter := r.URL.Query().Get("status")
	sortParam := r.URL.Query().Get("sort")

	// Validate query values if provided
	if trackFilter != "" {
		validTracks := map[string]bool{
			"Backend": true, "Frontend": true,
			"Product Design": true, "Product Management": true, "Growth": true,
		}
		if !validTracks[trackFilter] {
			writeError(w, http.StatusBadRequest, "invalid_query", "track must be one of: Backend, Frontend, Product Design, Product Management, Growth")
			return
		}
	}
	if statusFilter != "" {
		validStatuses := map[string]bool{
			"pending": true, "submitted": true, "reviewed": true,
		}
		if !validStatuses[statusFilter] {
			writeError(w, http.StatusBadRequest, "invalid_query", "status must be one of: pending, submitted, reviewed")
			return
		}
	}

	// Start with all checkins, then filter down
	result := []Checkin{}
	for _, item := range checkins {
		// Skip this item if it doesn't match the filter
		if trackFilter != "" && item.Track != trackFilter {
			continue
		}
		if statusFilter != "" && item.Status != statusFilter {
			continue
		}
		result = append(result, item)
	}

	// Sort if requested
	// We only support sort=submitted_at (ascending)
	if sortParam == "submitted_at" {
		sort.Slice(result, func(i, j int) bool {
			return result[i].SubmittedAt < result[j].SubmittedAt
		})
	} else if sortParam != "" {
		// Unknown sort field
		writeError(w, http.StatusBadRequest, "invalid_query", "sort only supports: submitted_at")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ---- GET /checkins/{id} ----
// Returns a single record by ID, or 404 if not found.
func GetCheckinByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	// id must not be empty (mux won't even match if it's missing, but just in case)
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "invalid_id", "id cannot be empty")
		return
	}

	for _, item := range checkins {
		if item.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(item)
			return
		}
	}

	// If we got here, nothing matched
	writeError(w, http.StatusNotFound, "not_found", "no checkin found with id "+id)
}

// ---- POST /checkins ----
// Creates a new checkin after validating the request body.
func CreateCheckin(w http.ResponseWriter, r *http.Request) {
	var newCheckin Checkin

	// Try to decode the JSON body
	if err := json.NewDecoder(r.Body).Decode(&newCheckin); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	// Validate all fields
	if errMsg := validateCheckin(newCheckin); errMsg != "" {
		writeError(w, http.StatusBadRequest, "validation_error", errMsg)
		return
	}

	// Check that the ID isn't already taken
	for _, item := range checkins {
		if item.ID == newCheckin.ID {
			writeError(w, http.StatusBadRequest, "duplicate_id", "a checkin with id "+newCheckin.ID+" already exists")
			return
		}
	}

	checkins = append(checkins, newCheckin)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created is more correct than 200 for POST
	json.NewEncoder(w).Encode(newCheckin)
}

// ---- PATCH /checkins/{id} ----
// Partially updates a checkin. Only fields sent in the body are changed.
func PatchCheckin(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	// Find the checkin first
	for index, item := range checkins {
		if item.ID == id {
			// Decode what the client sent
			var updatedData Checkin
			if err := json.NewDecoder(r.Body).Decode(&updatedData); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
				return
			}

			// Only update fields that were actually sent (non-empty)
			// This is what "partial update" means — untouched fields stay as-is
			if strings.TrimSpace(updatedData.LearnerName) != "" {
				checkins[index].LearnerName = updatedData.LearnerName
			}
			if strings.TrimSpace(updatedData.Track) != "" {
				// Validate before applying
				validTracks := map[string]bool{
					"Backend": true, "Frontend": true,
					"Product Design": true, "Product Management": true, "Growth": true,
				}
				if !validTracks[updatedData.Track] {
					writeError(w, http.StatusBadRequest, "validation_error", "track must be one of: Backend, Frontend, Product Design, Product Management, Growth")
					return
				}
				checkins[index].Track = updatedData.Track
			}
			if strings.TrimSpace(updatedData.Status) != "" {
				// Validate before applying
				validStatuses := map[string]bool{
					"pending": true, "submitted": true, "reviewed": true,
				}
				if !validStatuses[updatedData.Status] {
					writeError(w, http.StatusBadRequest, "validation_error", "status must be one of: pending, submitted, reviewed")
					return
				}
				checkins[index].Status = updatedData.Status
			}
			if strings.TrimSpace(updatedData.SubmittedAt) != "" {
				checkins[index].SubmittedAt = updatedData.SubmittedAt
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(checkins[index])
			return
		}
	}

	// Nothing matched
	writeError(w, http.StatusNotFound, "not_found", "no checkin found with id "+id)
}

// ---- DELETE /checkins/{id} ----
// Removes a record, or returns 404 if it doesn't exist.
func DeleteCheckin(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	for index, item := range checkins {
		if item.ID == id {
			// Remove by splicing the slice
			checkins = append(checkins[:index], checkins[index+1:]...)
			w.WriteHeader(http.StatusNoContent) // 204 = success, no body
			return
		}
	}

	writeError(w, http.StatusNotFound, "not_found", "no checkin found with id "+id)
}