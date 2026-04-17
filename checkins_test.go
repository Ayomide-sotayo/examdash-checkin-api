package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"github.com/gorilla/mux"
)

// resetCheckins gives us a fresh, predictable slice before each test.
// Without this, tests would interfere with each other.
func resetCheckins() {
	checkins = []Checkin{
		{ID: "1", LearnerName: "User One", Track: "Backend", Status: "submitted", SubmittedAt: time.Now().Format(time.RFC3339)},
		{ID: "2", LearnerName: "User Two", Track: "Frontend", Status: "pending", SubmittedAt: time.Now().Format(time.RFC3339)},
		{ID: "3", LearnerName: "User Three", Track: "Product Design", Status: "reviewed", SubmittedAt: time.Now().Format(time.RFC3339)},
	}
}

// --- Test 1: GET /checkins returns all records ---
func TestGetCheckins(t *testing.T) {
	resetCheckins()

	// httptest.NewRecorder() is a fake ResponseWriter that captures the response
	req := httptest.NewRequest("GET", "/checkins", nil)
	rr := httptest.NewRecorder()

	GetCheckins(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Decode the response body and count the records
	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 3 {
		t.Errorf("expected 3 checkins, got %d", len(result))
	}
}

// --- Test 2: POST /checkins creates a new record ---
func TestCreateCheckin(t *testing.T) {
	resetCheckins()

	newItem := Checkin{
		ID:          "4",
		LearnerName: "Test Learner",
		Track:       "Backend",
		Status:      "pending",
		SubmittedAt: time.Now().Format(time.RFC3339),
	}

	// Marshal the struct into JSON bytes to use as the request body
	body, _ := json.Marshal(newItem)
	req := httptest.NewRequest("POST", "/checkins", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	CreateCheckin(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	// The total list should now have 4 items
	if len(checkins) != 4 {
		t.Errorf("expected 4 checkins after POST, got %d", len(checkins))
	}
}

// --- Test 3: POST /checkins rejects invalid data ---
func TestCreateCheckin_ValidationError(t *testing.T) {
	resetCheckins()

	// learner_name is empty — should fail validation
	badItem := Checkin{
		ID:          "5",
		LearnerName: "",  // INVALID
		Track:       "Backend",
		Status:      "pending",
		SubmittedAt: time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(badItem)
	req := httptest.NewRequest("POST", "/checkins", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	CreateCheckin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid data, got %d", rr.Code)
	}

	// Check the response has the "error" key
	var errResponse map[string]string
	json.NewDecoder(rr.Body).Decode(&errResponse)
	if errResponse["error"] == "" {
		t.Error("expected an 'error' key in the response body")
	}
}

// --- Test 4: GET /checkins?track=Backend returns only matching records ---
func TestGetCheckins_FilterByTrack(t *testing.T) {
	resetCheckins()

	req := httptest.NewRequest("GET", "/checkins?track=Backend", nil)
	rr := httptest.NewRecorder()

	GetCheckins(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)

	// Only 1 of the 3 seeded records is Backend
	if len(result) != 1 {
		t.Errorf("expected 1 Backend checkin, got %d", len(result))
	}
	if result[0].Track != "Backend" {
		t.Errorf("expected track 'Backend', got '%s'", result[0].Track)
	}
}

// --- Test 5: DELETE /checkins/{id} returns 404 for unknown id ---
func TestDeleteCheckin_NotFound(t *testing.T) {
	resetCheckins()

	// We need mux vars for {id} — set them up manually for testing
	req := httptest.NewRequest("DELETE", "/checkins/999", nil)
	req = setMuxVars(req, map[string]string{"id": "999"})
	rr := httptest.NewRecorder()

	DeleteCheckin(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

// setMuxVars is a small helper that injects URL vars into a request for testing.
// Normally mux does this automatically, but in unit tests we call handlers directly.
func setMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}