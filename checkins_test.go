package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// setupTestDB connects to the real DB for integration tests.
// Set TEST_DATABASE_URL (or DATABASE_URL) before running.
// If neither is set, the test is skipped — so CI without a DB still passes.
func setupTestDB(t *testing.T) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("No DATABASE_URL set — skipping DB integration tests")
	}

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil || DB.Ping() != nil {
		t.Skipf("Could not connect to test DB: %v", err)
	}
}

// seedTestData wipes and re-inserts a clean set of rows before each test.
func seedTestData(t *testing.T) {
	t.Helper()
	DB.Exec(`DELETE FROM checkins`)
	DB.Exec(`DELETE FROM tracks`)
	DB.Exec(`INSERT INTO tracks (name) VALUES ('Backend'),('Frontend'),('Product Design'),('Product Management'),('Growth') ON CONFLICT DO NOTHING`)
	DB.Exec(`
		INSERT INTO checkins (learner_name, track_id, status, submitted_at, created_at, updated_at)
		SELECT 'User One', t.id, 'submitted', $1, NOW(), NOW() FROM tracks t WHERE t.name='Backend'`,
		time.Now().Format(time.RFC3339))
	DB.Exec(`
		INSERT INTO checkins (learner_name, track_id, status, submitted_at, created_at, updated_at)
		SELECT 'User Two', t.id, 'pending', $1, NOW(), NOW() FROM tracks t WHERE t.name='Frontend'`,
		time.Now().Format(time.RFC3339))
	DB.Exec(`
		INSERT INTO checkins (learner_name, track_id, status, submitted_at, created_at, updated_at)
		SELECT 'User Three', t.id, 'reviewed', $1, NOW(), NOW() FROM tracks t WHERE t.name='Product Design'`,
		time.Now().Format(time.RFC3339))
}

// setMuxVars injects URL path vars for direct handler testing.
func setMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

// getFirstCheckinID is a helper to grab the first checkin id from the DB
func getFirstCheckinID(t *testing.T) int {
	t.Helper()
	var id int
	err := DB.QueryRow(`SELECT id FROM checkins ORDER BY id ASC LIMIT 1`).Scan(&id)
	if err != nil {
		t.Fatalf("could not get first checkin id: %v", err)
	}
	return id
}

// --- Test 1: GET /checkins returns all records ---
func TestGetCheckins(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	req := httptest.NewRequest("GET", "/checkins", nil)
	rr := httptest.NewRecorder()
	GetCheckins(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 3 {
		t.Errorf("expected 3 checkins, got %d", len(result))
	}
}

// --- Test 2: POST /checkins creates a new record ---
func TestCreateCheckin(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	// No id field — Postgres generates it automatically
	newItem := Checkin{
		LearnerName: "Test Learner",
		Track:       "Backend",
		Status:      "pending",
		SubmittedAt: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(newItem)
	req := httptest.NewRequest("POST", "/checkins", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	CreateCheckin(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	// Verify the returned checkin has an auto-generated id > 0
	var created Checkin
	json.NewDecoder(rr.Body).Decode(&created)
	if created.ID == 0 {
		t.Error("expected auto-generated id > 0, got 0")
	}
}

// --- Test 3: POST /checkins rejects missing learner_name ---
func TestCreateCheckin_ValidationError(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	badItem := Checkin{
		LearnerName: "", // INVALID
		Track:       "Backend",
		Status:      "pending",
		SubmittedAt: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(badItem)
	req := httptest.NewRequest("POST", "/checkins", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	CreateCheckin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var errResp map[string]string
	json.NewDecoder(rr.Body).Decode(&errResp)
	if errResp["error"] == "" {
		t.Error("expected 'error' key in response body")
	}
}

// --- Test 4: GET /checkins?track=Backend returns only matching records ---
func TestGetCheckins_FilterByTrack(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	req := httptest.NewRequest("GET", "/checkins?track=Backend", nil)
	rr := httptest.NewRecorder()
	GetCheckins(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("expected 1 Backend checkin, got %d", len(result))
	}
	if len(result) > 0 && result[0].Track != "Backend" {
		t.Errorf("expected track 'Backend', got '%s'", result[0].Track)
	}
}

// --- Test 5: DELETE /checkins/{id} returns 404 for unknown id ---
func TestDeleteCheckin_NotFound(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	req := httptest.NewRequest("DELETE", "/checkins/999999", nil)
	req = setMuxVars(req, map[string]string{"id": "999999"})
	rr := httptest.NewRecorder()
	DeleteCheckin(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// --- Test 6: GET /checkins?page=1&limit=2 returns paginated results ---
func TestGetCheckins_Pagination(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	req := httptest.NewRequest("GET", "/checkins?page=1&limit=2", nil)
	rr := httptest.NewRecorder()
	GetCheckins(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 2 {
		t.Errorf("expected 2 checkins with limit=2, got %d", len(result))
	}
}