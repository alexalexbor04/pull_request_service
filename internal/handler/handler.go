package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/alexalexbor04/pull_request_service/internal/entities"
	"github.com/alexalexbor04/pull_request_service/internal/service"
)

type Handler struct {
	service *service.Service
}

func New(service *service.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /team/add", h.AddTeam)
	mux.HandleFunc("GET /team/get", h.GetTeam)

	mux.HandleFunc("POST /users/setIsActive", h.SetUserActive)
	mux.HandleFunc("GET /users/getReview", h.GetUserReviews)

	mux.HandleFunc("POST /pullRequest/create", h.CreatePullRequest)
	mux.HandleFunc("POST /pullRequest/merge", h.MergePullRequest)
	mux.HandleFunc("POST /pullRequest/reassign", h.ReassignReviewer)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, entities.ErrorResponse{
		Error: entities.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func (h *Handler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var team entities.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}
	
	if err := h.service.CreateTeam(&team); err != nil {
		if err.Error() == entities.ErrTeamExists {
			writeError(w, http.StatusBadRequest, entities.ErrTeamExists, "team_name already exists")
			return
		}
		log.Printf("Error creating team: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	createdTeam, err := h.service.GetTeam(team.TeamName)
	if err != nil {
		log.Printf("Error getting created team: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"team": createdTeam,
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name is required")
		return
	}

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "team not found")
			return
		}
		log.Printf("Error getting team: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, team)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	user, err := h.service.SetUserActive(req.UserID, req.IsActive)
	if err != nil {
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "user not found")
			return
		}
		log.Printf("Error setting user active: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id is required")
		return
	}

	prs, err := h.service.GetUserReviews(userID)
	if err != nil {
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "user not found")
			return
		}
		log.Printf("Error getting user reviews: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	if prs == nil {
		prs = []entities.PullRequestShort{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

func (h *Handler) CreatePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.CreatePullRequest(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		if err.Error() == entities.ErrPRExists {
			writeError(w, http.StatusConflict, entities.ErrPRExists, "PR id already exists")
			return
		}
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "author not found")
			return
		}
		log.Printf("Error creating PR: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) MergePullRequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.MergePullRequest(req.PullRequestID)
	if err != nil {
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "PR not found")
			return
		}
		log.Printf("Error merging PR: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	pr, newReviewerID, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		if err.Error() == entities.ErrNotFound {
			writeError(w, http.StatusNotFound, entities.ErrNotFound, "PR or user not found")
			return
		}
		if err.Error() == entities.ErrPRMerged {
			writeError(w, http.StatusConflict, entities.ErrPRMerged, "cannot reassign on merged PR")
			return
		}
		if err.Error() == entities.ErrNotAssigned {
			writeError(w, http.StatusConflict, entities.ErrNotAssigned, "reviewer is not assigned to this PR")
			return
		}
		if err.Error() == entities.ErrNoCandidate {
			writeError(w, http.StatusConflict, entities.ErrNoCandidate, "no active replacement candidate in team")
			return
		}
		log.Printf("Error reassigning reviewer: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": newReviewerID,
	})
}



