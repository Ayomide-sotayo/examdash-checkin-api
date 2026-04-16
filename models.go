package main

import "time"


type Checkin struct {
	ID          string    `json:"id"`
	LearnerName string    `json:"learner_name"`
	Track       string    `json:"track"` 
	Status      string    `json:"status"` 
	SubmittedAt string    `json:"submitted_at"`
}


var checkins = []Checkin{
	{ID: "1", LearnerName: "User One", Track: "Backend", Status: "submitted", SubmittedAt: time.Now().Format(time.RFC3339)},
	{ID: "2", LearnerName: "User Two", Track: "Frontend", Status: "pending", SubmittedAt: time.Now().Format(time.RFC3339)},
	{ID: "3", LearnerName: "User Three", Track: "Product Design", Status: "reviewed", SubmittedAt: time.Now().Format(time.RFC3339)},
}