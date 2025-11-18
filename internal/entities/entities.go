package entities

import "time"

type User struct {
	ID string `json:"user_id" db:"id"`
	Username string `json:"username" db:"username"`
	TeamName string `json:"team_name" db:"team_name"`
	IsActive bool `json:"is_active" db:"is_active"`
}

type TeamMember struct {
	UserID string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool `json:"is_active"`
}

type Team struct {
	TeamName string `json:"team_name"`
	Members []TeamMember `json:"members"`
}

type PullRequest struct {
	ID string `json:"pull_request_id" db:"id"`
	Name string `json:"pull_request_name" db:"name"`
	AuthorID string `json:"author_id" db:"author_id"`
	Status string `json:"status" db:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt *time.Time `json:"createdAt,omitempty" db:"created_at"`
	MergedAt *time.Time `json:"mergedAt,omitempty" db:"merged_at"`
}

type PullRequestShort struct {
	ID string `json:"pull_request_id"`
	Name string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

const (
	StatusOpen = "OPEN"
	StatusMerged = "MERGED"
)

const (
	ErrTeamExists = "TEAM_EXISTS"
	ErrPRExists = "PR_EXISTS"
	ErrPRMerged = "PR_MERGED"
	ErrNotAssigned = "NOT_ASSIGNED"
	ErrNoCandidate = "NO_CANDIDATE"
	ErrNotFound = "NOT_FOUND"
)