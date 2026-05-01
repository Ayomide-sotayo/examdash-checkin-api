package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ---- Helper: write a JSON error response ----
func writeError(w http.ResponseWriter, statusCode int, errType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   errType,
		"message": message,
	})
}

// ---- Valid enum maps (used in validation + query checks) ----
var validTracks = map[string]bool{
	"Backend": true, "Frontend": true,
	"Product Design": true, "Product Management": true, "Growth": true,
}
var validStatuses = map[string]bool{
	"pending": true, "submitted": true, "reviewed": true,
}

// ---- Helper: validate a Checkin struct before write ----
func validateCheckin(c Checkin) string {
	if strings.TrimSpace(c.LearnerName) == "" {
		return "learner_name is required and cannot be empty"
	}
	if !validTracks[c.Track] {
		return "track must be one of: Backend, Frontend, Product Design, Product Management, Growth"
	}
	if !validStatuses[c.Status] {
		return "status must be one of: pending, submitted, reviewed"
	}
	if strings.TrimSpace(c.SubmittedAt) == "" {
		return "submitted_at is required"
	}
	return ""
}

// ---- GET /checkins ----
// Supports: ?track=Backend  ?status=pending  ?sort=submitted_at  ?page=1&limit=5
func GetCheckins(w http.ResponseWriter, r *http.Request) {
	trackFilter  := r.URL.Query().Get("track")
	statusFilter := r.URL.Query().Get("status")
	sortParam    := r.URL.Query().Get("sort")
	pageStr      := r.URL.Query().Get("page")
	limitStr     := r.URL.Query().Get("limit")

	// Validate filters
	if trackFilter != "" && !validTracks[trackFilter] {
		writeError(w, http.StatusBadRequest, "invalid_query", "track must be one of: Backend, Frontend, Product Design, Product Management, Growth")
		return
	}
	if statusFilter != "" && !validStatuses[statusFilter] {
		writeError(w, http.StatusBadRequest, "invalid_query", "status must be one of: pending, submitted, reviewed")
		return
	}
	if sortParam != "" && sortParam != "submitted_at" {
		writeError(w, http.StatusBadRequest, "invalid_query", "sort only supports: submitted_at")
		return
	}

	// Pagination defaults
	page, limit := 1, 10
	if pageStr != "" {
		if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
			page = v
		}
	}
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := (page - 1) * limit

	// Build the SQL query dynamically based on filters
	query := `
		SELECT c.id, c.learner_name, t.name AS track, c.status,
		       c.submitted_at, c.created_at, c.updated_at
		FROM checkins c
		JOIN tracks t ON c.track_id = t.id
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if trackFilter != "" {
		query += fmt.Sprintf(" AND t.name = $%d", argIdx)
		args = append(args, trackFilter)
		argIdx++
	}
	if statusFilter != "" {
		query += fmt.Sprintf(" AND c.status = $%d", argIdx)
		args = append(args, statusFilter)
		argIdx++
	}

	// Sort
	if sortParam == "submitted_at" {
		query += " ORDER BY c.submitted_at ASC"
	} else {
		query += " ORDER BY c.created_at DESC"
	}

	// Pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to query checkins")
		return
	}
	defer rows.Close()

	result := []Checkin{}
	for rows.Next() {
		var c Checkin
		if err := rows.Scan(&c.ID, &c.LearnerName, &c.Track, &c.Status,
			&c.SubmittedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "failed to read checkins")
			return
		}
		result = append(result, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ---- GET /checkins/{id} ----
func GetCheckinByID(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]

	// id must be a valid integer now
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var c Checkin
	err = DB.QueryRow(`
		SELECT c.id, c.learner_name, t.name, c.status,
		       c.submitted_at, c.created_at, c.updated_at
		FROM checkins c
		JOIN tracks t ON c.track_id = t.id
		WHERE c.id = $1`, id).
		Scan(&c.ID, &c.LearnerName, &c.Track, &c.Status,
			&c.SubmittedAt, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

// ---- POST /checkins ----
// Caller does NOT send an id — Postgres generates it automatically
func CreateCheckin(w http.ResponseWriter, r *http.Request) {
	var newCheckin Checkin
	if err := json.NewDecoder(r.Body).Decode(&newCheckin); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	if errMsg := validateCheckin(newCheckin); errMsg != "" {
		writeError(w, http.StatusBadRequest, "validation_error", errMsg)
		return
	}

	// Look up the track_id from the tracks table
	var trackID int
	err := DB.QueryRow(`SELECT id FROM tracks WHERE name = $1`, newCheckin.Track).Scan(&trackID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_track", "track not found: "+newCheckin.Track)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Use RETURNING id to get the auto-generated id back
	err = DB.QueryRow(`
		INSERT INTO checkins (learner_name, track_id, status, submitted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		newCheckin.LearnerName, trackID,
		newCheckin.Status, newCheckin.SubmittedAt, now, now).Scan(&newCheckin.ID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to create checkin")
		return
	}

	newCheckin.CreatedAt = now
	newCheckin.UpdatedAt = now

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newCheckin)
}

// ---- PATCH /checkins/{id} ----
func PatchCheckin(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	// Check the record exists first
	var existing Checkin
	var trackID int
	err = DB.QueryRow(`
		SELECT c.id, c.learner_name, t.name, t.id, c.status, c.submitted_at
		FROM checkins c JOIN tracks t ON c.track_id = t.id
		WHERE c.id = $1`, id).
		Scan(&existing.ID, &existing.LearnerName, &existing.Track, &trackID,
			&existing.Status, &existing.SubmittedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	var patch Checkin
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	// Merge: only overwrite fields that were sent
	if strings.TrimSpace(patch.LearnerName) != "" {
		existing.LearnerName = patch.LearnerName
	}
	if strings.TrimSpace(patch.Track) != "" {
		if !validTracks[patch.Track] {
			writeError(w, http.StatusBadRequest, "validation_error", "track must be one of: Backend, Frontend, Product Design, Product Management, Growth")
			return
		}
		existing.Track = patch.Track
		DB.QueryRow(`SELECT id FROM tracks WHERE name = $1`, existing.Track).Scan(&trackID)
	}
	if strings.TrimSpace(patch.Status) != "" {
		if !validStatuses[patch.Status] {
			writeError(w, http.StatusBadRequest, "validation_error", "status must be one of: pending, submitted, reviewed")
			return
		}
		existing.Status = patch.Status
	}
	if strings.TrimSpace(patch.SubmittedAt) != "" {
		existing.SubmittedAt = patch.SubmittedAt
	}

	now := time.Now().UTC().Format(time.RFC3339)

	_, err = DB.Exec(`
		UPDATE checkins
		SET learner_name=$1, track_id=$2, status=$3, submitted_at=$4, updated_at=$5
		WHERE id=$6`,
		existing.LearnerName, trackID, existing.Status, existing.SubmittedAt, now, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to update checkin")
		return
	}

	existing.UpdatedAt = now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existing)
}

// ---- DELETE /checkins/{id} ----
func DeleteCheckin(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	result, err := DB.Exec(`DELETE FROM checkins WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to delete checkin")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}