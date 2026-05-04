package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
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

// ---- Valid enum maps ----
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

// ---- POST /auth/signup ----
func Signup(w http.ResponseWriter, r *http.Request) {
	var input User
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	if strings.TrimSpace(input.Email) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "email is required")
		return
	}
	if strings.TrimSpace(input.Password) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "password is required")
		return
	}

	// Hash the password before saving
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to hash password")
		return
	}

	var userID int
	// Default to learner if no role specified
role := input.Role
if role == "" {
    role = "learner"
}
if role != "learner" && role != "reviewer" {
    writeError(w, http.StatusBadRequest, "validation_error", "role must be either learner or reviewer")
    return
}

err = DB.QueryRow(`
    INSERT INTO users (email, password, role)
    VALUES ($1, $2, $3)
    RETURNING id`,
    input.Email, string(hashed), role).Scan(&userID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusBadRequest, "duplicate_email", "a user with that email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "db_error", "failed to create user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
json.NewEncoder(w).Encode(map[string]interface{}{
    "id":    userID,
    "email": input.Email,
    "role":  role,
})
}

// ---- POST /auth/login ----
func Login(w http.ResponseWriter, r *http.Request) {
	var input User
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	// Find the user by email
	var user User
	err := DB.QueryRow(`
		SELECT id, email, password, role FROM users WHERE email = $1`,
		input.Email).Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	// Compare hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	// Generate JWT token — expires in 24 hours
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	tokenStr, err := generateToken(claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "failed to generate token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenStr,
		"role":  user.Role,
	})
}

// ---- GET /checkins ----
// Learners see only their own checkins
// Reviewers see all checkins
func GetCheckins(w http.ResponseWriter, r *http.Request) {
	claims := getClaimsFromContext(r)

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

	query := `
		SELECT c.id, c.user_id, c.learner_name, t.name AS track, c.status,
		       c.submitted_at, c.created_at, c.updated_at
		FROM checkins c
		JOIN tracks t ON c.track_id = t.id
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	// Ownership rule — learners only see their own checkins
	if claims.Role == "learner" {
		query += fmt.Sprintf(" AND c.user_id = $%d", argIdx)
		args = append(args, claims.UserID)
		argIdx++
	}

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

	if sortParam == "submitted_at" {
		query += " ORDER BY c.submitted_at ASC"
	} else {
		query += " ORDER BY c.created_at DESC"
	}

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
		if err := rows.Scan(&c.ID, &c.UserID, &c.LearnerName, &c.Track, &c.Status,
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
	claims := getClaimsFromContext(r)
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var c Checkin
	err = DB.QueryRow(`
		SELECT c.id, c.user_id, c.learner_name, t.name, c.status,
		       c.submitted_at, c.created_at, c.updated_at
		FROM checkins c
		JOIN tracks t ON c.track_id = t.id
		WHERE c.id = $1`, id).
		Scan(&c.ID, &c.UserID, &c.LearnerName, &c.Track, &c.Status,
			&c.SubmittedAt, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	// Ownership rule — learner can only view their own checkin
	if claims.Role == "learner" && c.UserID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this checkin")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

// ---- POST /checkins ----
func CreateCheckin(w http.ResponseWriter, r *http.Request) {
	claims := getClaimsFromContext(r)

	var newCheckin Checkin
	if err := json.NewDecoder(r.Body).Decode(&newCheckin); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

	if errMsg := validateCheckin(newCheckin); errMsg != "" {
		writeError(w, http.StatusBadRequest, "validation_error", errMsg)
		return
	}

	var trackID int
	err := DB.QueryRow(`SELECT id FROM tracks WHERE name = $1`, newCheckin.Track).Scan(&trackID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_track", "track not found: "+newCheckin.Track)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Always assign the checkin to the logged-in user
	err = DB.QueryRow(`
		INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		claims.UserID, newCheckin.LearnerName, trackID,
		newCheckin.Status, newCheckin.SubmittedAt, now, now).Scan(&newCheckin.ID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to create checkin")
		return
	}

	newCheckin.UserID    = claims.UserID
	newCheckin.CreatedAt = now
	newCheckin.UpdatedAt = now

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newCheckin)
}

// ---- PATCH /checkins/{id} ----
func PatchCheckin(w http.ResponseWriter, r *http.Request) {
	claims := getClaimsFromContext(r)
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	var existing Checkin
	var trackID int
	err = DB.QueryRow(`
		SELECT c.id, c.user_id, c.learner_name, t.name, t.id, c.status, c.submitted_at
		FROM checkins c JOIN tracks t ON c.track_id = t.id
		WHERE c.id = $1`, id).
		Scan(&existing.ID, &existing.UserID, &existing.LearnerName,
			&existing.Track, &trackID, &existing.Status, &existing.SubmittedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	// Ownership rule — learner can only update their own checkin
	if claims.Role == "learner" && existing.UserID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this checkin")
		return
	}

	var patch Checkin
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}

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
	claims := getClaimsFromContext(r)
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a number")
		return
	}

	// Check it exists first
	var ownerID int
	err = DB.QueryRow(`SELECT user_id FROM checkins WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("no checkin found with id %d", id))
		return
	}

	// Ownership rule — learner can only delete their own checkin
	if claims.Role == "learner" && ownerID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden", "you do not have access to this checkin")
		return
	}

	_, err = DB.Exec(`DELETE FROM checkins WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "failed to delete checkin")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}