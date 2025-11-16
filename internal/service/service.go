package service

import (
	"database/sql"
	"errors"
	"math/rand"
	"time"

	"github.com/alexalexbor04/pull_request_service/internal/entities"
	"github.com/alexalexbor04/pull_request_service/internal/repos"
)

type Service struct {
	repo *repos.Repo
	rand *rand.Rand
}

func New(repo *repos.Repo) *Service {
	return &Service{
		repo: repo,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) CreateTeam(team *entities.Team) error {
	exists, err := s.repo.TeamExists(team.TeamName)
	if err != nil {
		return err
	}
	if exists {
		return errors.New(entities.ErrTeamExists)
	}

	if err := s.repo.CreateTeam(team.TeamName); err != nil {
		return err
	}

	for _, member := range team.Members {
		user := &entities.User{
			ID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := s.repo.CreateOrUpdateUser(user); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetTeam(teamName string) (*entities.Team, error) {
	team, err := s.repo.GetTeam(teamName)
	if err == sql.ErrNoRows {
		return nil, errors.New(entities.ErrNotFound)
	}
	return team, err
}

func (s *Service) SetUserActive(userID string, isActive bool) (*entities.User, error) {
	user, err := s.repo.GetUser(userID)
	if err == sql.ErrNoRows {
		return nil, errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	if err := s.repo.SetUserActive(userID, isActive); err != nil {
		return nil, err
	}

	user.IsActive = isActive
	return user, nil
}

func (s *Service) CreatePullRequest(prID, prName, authorID string) (*entities.PullRequest, error) {
	exists, err := s.repo.PRExists(prID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New(entities.ErrPRExists)
	}

	author, err := s.repo.GetUser(authorID)
	if err == sql.ErrNoRows {
		return nil, errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	candidates, err := s.repo.GetActiveTeamMembers(author.TeamName, []string{authorID})
	if err != nil {
		return nil, err
	}

	reviewers := s.selectRandomReviewers(candidates, 2)
	reviewerIDs := make([]string, len(reviewers))
	for i, r := range reviewers {
		reviewerIDs[i] = r.ID
	}
	
	pr := &entities.PullRequest{
		ID:     prID,
		Name:   prName,
		AuthorID:          authorID,
		Status:            entities.StatusOpen,
		AssignedReviewers: reviewerIDs,
	}

	if err := s.repo.CreatePullRequest(pr, reviewerIDs); err != nil {
		return nil, err
	}

	return s.repo.GetPullRequest(prID)
}

func (s *Service) selectRandomReviewers(candidates []entities.User, maxCount int) []entities.User {
	if len(candidates) == 0 {
		return []entities.User{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	shuffled := make([]entities.User, len(candidates))
	copy(shuffled, candidates)
	s.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}

func (s *Service) MergePullRequest(prID string) (*entities.PullRequest, error) {
	pr, err := s.repo.GetPullRequest(prID)
	if err == sql.ErrNoRows {
		return nil, errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	if pr.Status == entities.StatusMerged {
		return pr, nil
	}

	now := time.Now()
	if err := s.repo.UpdatePRStatus(prID, entities.StatusMerged, &now); err != nil {
		return nil, err
	}

	return s.repo.GetPullRequest(prID)
}

func (s *Service) ReassignReviewer(prID, oldUserID string) (*entities.PullRequest, string, error) {
	pr, err := s.repo.GetPullRequest(prID)
	if err == sql.ErrNoRows {
		return nil, "", errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, "", err
	}

	if pr.Status == entities.StatusMerged {
		return nil, "", errors.New(entities.ErrPRMerged)
	}

	isAssigned := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldUserID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return nil, "", errors.New(entities.ErrNotAssigned)
	}

	oldUser, err := s.repo.GetUser(oldUserID)
	if err == sql.ErrNoRows {
		return nil, "", errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, "", err
	}

	excludeIDs := append([]string{pr.AuthorID}, pr.AssignedReviewers...)

	candidates, err := s.repo.GetActiveTeamMembers(oldUser.TeamName, excludeIDs)
	if err != nil {
		return nil, "", err
	}

	if len(candidates) == 0 {
		return nil, "", errors.New(entities.ErrNoCandidate)
	}

	newReviewer := s.selectRandomReviewers(candidates, 1)[0]

	if err := s.repo.ReplaceReviewer(prID, oldUserID, newReviewer.ID); err != nil {
		return nil, "", err
	}

	updatedPR, err := s.repo.GetPullRequest(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewer.ID, nil
}

func (s *Service) GetUserReviews(userID string) ([]entities.PullRequestShort, error) {
	_, err := s.repo.GetUser(userID)
	if err == sql.ErrNoRows {
		return nil, errors.New(entities.ErrNotFound)
	}
	if err != nil {
		return nil, err
	}

	return s.repo.GetUserReviews(userID)
}


