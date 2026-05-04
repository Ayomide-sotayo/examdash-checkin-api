package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ---- Test DB Setup ----
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

// ---- Seed clean data before each test ----
func seedTestData(t *testing.T) {
	t.Helper()
	DB.Exec(`DELETE FROM checkins`)
	DB.Exec(`DELETE FROM users`)
	DB.Exec(`DELETE FROM tracks`)
	DB.Exec(`INSERT INTO tracks (name) VALUES ('Backend'),('Frontend'),('Product Design'),('Product Management'),('Growth') ON CONFLICT DO NOTHING`)
}

// ---- Helper: create a test user and return their id ----
func createTestUser(t *testing.T, email, password, role string) int {
	t.Helper()
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	var id int
	err := DB.QueryRow(`
		INSERT INTO users (email, password, role)
		VALUES ($1, $2, $3)
		RETURNING id`, email, string(hashed), role).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

// ---- Helper: create a test checkin and return its id ----
func createTestCheckin(t *testing.T, userID int, track, status string) int {
	t.Helper()
	var trackID int
	DB.QueryRow(`SELECT id FROM tracks WHERE name = $1`, track).Scan(&trackID)
	var id int
	err := DB.QueryRow(`
		INSERT INTO checkins (user_id, learner_name, track_id, status, submitted_at, created_at, updated_at)
		VALUES ($1, 'Test Learner', $2, $3, $4, NOW(), NOW())
		RETURNING id`,
		userID, trackID, status, time.Now().Format(time.RFC3339)).Scan(&id)
	if err != nil {
		t.Fatalf("failed to create test checkin: %v", err)
	}
	return id
}

// ---- Helper: generate a real JWT token for a test user ----
func generateTestToken(t *testing.T, userID int, email, role string) string {
	t.Helper()
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}
	import_jwt_token, err := generateToken(claims)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return import_jwt_token
}

// ---- Helper: inject mux vars ----
func setMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

// ---- Helper: make an authenticated request ----
func makeAuthRequest(method, url, token string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// ============================================================
// AUTH TESTS
// ============================================================

// --- Test 1: POST /auth/signup creates a new user ---
func TestSignup(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	Signup(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)
	if result["email"] != "test@example.com" {
		t.Errorf("expected email in response, got %v", result)
	}
}

// --- Test 2: POST /auth/signup rejects duplicate email ---
func TestSignup_DuplicateEmail(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)
	createTestUser(t, "dupe@example.com", "password123", "learner")

	body, _ := json.Marshal(map[string]string{
		"email":    "dupe@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	Signup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate email, got %d", rr.Code)
	}
}

// --- Test 3: POST /auth/login returns a token ---
func TestLogin(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)
	createTestUser(t, "login@example.com", "password123", "learner")

	body, _ := json.Marshal(map[string]string{
		"email":    "login@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var result map[string]string
	json.NewDecoder(rr.Body).Decode(&result)
	if result["token"] == "" {
		t.Error("expected token in response")
	}
}

// --- Test 4: POST /auth/login rejects wrong password ---
func TestLogin_WrongPassword(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)
	createTestUser(t, "wrong@example.com", "correctpassword", "learner")

	body, _ := json.Marshal(map[string]string{
		"email":    "wrong@example.com",
		"password": "wrongpassword",
	})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// --- Test 5: Protected route rejects request with no token ---
func TestProtectedRoute_NoToken(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	req := httptest.NewRequest("GET", "/checkins", nil)
	rr := httptest.NewRecorder()
	AuthMiddleware(GetCheckins)(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ============================================================
// OWNERSHIP TESTS
// ============================================================

// --- Test 6: Learner can only see their own checkins ---
func TestGetCheckins_LearnerOwnership(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	// Create two learners
	user1ID := createTestUser(t, "learner1@example.com", "pass123", "learner")
	user2ID := createTestUser(t, "learner2@example.com", "pass123", "learner")

	// Each learner has one checkin
	createTestCheckin(t, user1ID, "Backend", "pending")
	createTestCheckin(t, user2ID, "Frontend", "pending")

	// Learner 1 requests checkins
	token := generateTestToken(t, user1ID, "learner1@example.com", "learner")
	req := makeAuthRequest("GET", "/checkins", token, nil)
	rr := httptest.NewRecorder()
	AuthMiddleware(GetCheckins)(rr, req)

	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)

	// Learner 1 should only see their own 1 checkin
	if len(result) != 1 {
		t.Errorf("expected 1 checkin for learner, got %d", len(result))
	}
}

// --- Test 7: Reviewer can see all checkins ---
func TestGetCheckins_ReviewerSeesAll(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	user1ID := createTestUser(t, "learner3@example.com", "pass123", "learner")
	user2ID := createTestUser(t, "learner4@example.com", "pass123", "learner")
	reviewerID := createTestUser(t, "reviewer@example.com", "pass123", "reviewer")

	createTestCheckin(t, user1ID, "Backend", "pending")
	createTestCheckin(t, user2ID, "Frontend", "pending")

	token := generateTestToken(t, reviewerID, "reviewer@example.com", "reviewer")
	req := makeAuthRequest("GET", "/checkins", token, nil)
	rr := httptest.NewRecorder()
	AuthMiddleware(GetCheckins)(rr, req)

	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)

	// Reviewer should see all 2 checkins
	if len(result) != 2 {
		t.Errorf("expected 2 checkins for reviewer, got %d", len(result))
	}
}

// --- Test 8: Learner cannot access another learner's checkin ---
func TestGetCheckinByID_ForbiddenForOtherLearner(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	user1ID := createTestUser(t, "learner5@example.com", "pass123", "learner")
	user2ID := createTestUser(t, "learner6@example.com", "pass123", "learner")

	// User 1 creates a checkin
	checkinID := createTestCheckin(t, user1ID, "Backend", "pending")

	// User 2 tries to access it
	token := generateTestToken(t, user2ID, "learner6@example.com", "learner")
	req := makeAuthRequest("GET", fmt.Sprintf("/checkins/%d", checkinID), token, nil)
	req = setMuxVars(req, map[string]string{"id": fmt.Sprintf("%d", checkinID)})
	rr := httptest.NewRecorder()
	AuthMiddleware(GetCheckinByID)(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// --- Test 9: GET /checkins returns all records for reviewer ---
func TestGetCheckins(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	userID := createTestUser(t, "learner7@example.com", "pass123", "learner")
	reviewerID := createTestUser(t, "reviewer2@example.com", "pass123", "reviewer")

	createTestCheckin(t, userID, "Backend", "submitted")
	createTestCheckin(t, userID, "Frontend", "pending")
	createTestCheckin(t, userID, "Growth", "reviewed")

	token := generateTestToken(t, reviewerID, "reviewer2@example.com", "reviewer")
	req := makeAuthRequest("GET", "/checkins", token, nil)
	rr := httptest.NewRecorder()
	AuthMiddleware(GetCheckins)(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var result []Checkin
	json.NewDecoder(rr.Body).Decode(&result)
	if len(result) != 3 {
		t.Errorf("expected 3 checkins, got %d", len(result))
	}
}

// --- Test 10: POST /checkins creates checkin for logged in user ---
func TestCreateCheckin(t *testing.T) {
	setupTestDB(t)
	seedTestData(t)

	userID := createTestUser(t, "learner8@example.com", "pass123", "learner")
	token := generateTestToken(t, userID, "learner8@example.com", "learner")

	body, _ := json.Marshal(Checkin{
		LearnerName: "Test Learner",
		Track:       "Backend",
		Status:      "pending",
		SubmittedAt: time.Now().Format(time.RFC3339),
	})

	req := makeAuthRequest("POST", "/checkins", token, body)
	rr := httptest.NewRecorder()
	AuthMiddleware(CreateCheckin)(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var created Checkin
	json.NewDecoder(rr.Body).Decode(&created)
	if created.ID == 0 {
		t.Error("expected auto-generated id > 0")
	}
	if created.UserID != userID {
		t.Errorf("expected user_id %d, got %d", userID, created.UserID)
	}
}