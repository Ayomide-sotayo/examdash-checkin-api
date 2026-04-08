package main

import "time"

// Checkin represents the learner check-in data contract
type Checkin struct {
	ID          string    `json:"id"`
	LearnerName string    `json:"learner_name"`
	Track       string    `json:"track"` // e.g., Backend, Frontend [cite: 45]
	Status      string    `json:"status"` // pending, submitted, or reviewed 
	SubmittedAt string    `json:"submitted_at"`
}

// In-memory storage
var checkins = []Checkin{
	// Seeded data 
	{ID: "1", LearnerName: "User One", Track: "Backend", Status: "submitted", SubmittedAt: time.Now().Format(time.RFC3339)},
	{ID: "2", LearnerName: "User Two", Track: "Frontend", Status: "pending", SubmittedAt: time.Now().Format(time.RFC3339)},
	{ID: "3", LearnerName: "User Three", Track: "Product Design", Status: "reviewed", SubmittedAt: time.Now().Format(time.RFC3339)},
}