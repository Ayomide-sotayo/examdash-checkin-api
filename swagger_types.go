package main

// SignupInput is the request body for POST /auth/signup
type SignupInput struct {
	Email    string `json:"email" example:"ada@examdash.com"`
	Password string `json:"password" example:"password123"`
	Role     string `json:"role" example:"learner" enums:"learner,reviewer"`
}

// LoginInput is the request body for POST /auth/login
type LoginInput struct {
	Email    string `json:"email" example:"ada@examdash.com"`
	Password string `json:"password" example:"password123"`
}

// SignupResponse is the success response for POST /auth/signup
type SignupResponse struct {
	ID    int    `json:"id" example:"1"`
	Email string `json:"email" example:"ada@examdash.com"`
	Role  string `json:"role" example:"learner"`
}

// LoginResponse is the success response for POST /auth/login
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Role  string `json:"role" example:"learner"`
}

// CheckinInput is the request body for POST /checkins and PATCH /checkins/{id}
type CheckinInput struct {
	LearnerName string `json:"learner_name" example:"Ada Okafor"`
	Track       string `json:"track" example:"Backend" enums:"Backend,Frontend,Product Design,Product Management,Growth"`
	Status      string `json:"status" example:"pending" enums:"pending,submitted,reviewed"`
	SubmittedAt string `json:"submitted_at" example:"2026-05-08T10:00:00Z"`
}

// ErrorResponse is the standard error shape returned by all error responses
type ErrorResponse struct {
	Error   string `json:"error" example:"validation_error"`
	Message string `json:"message" example:"learner_name is required and cannot be empty"`
}