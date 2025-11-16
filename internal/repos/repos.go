package repos

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alexalexbor04/pull_request_service/internal/entities"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo { 
	return &Repo{db: db}
}

func (r *Repo) CreateTeam(teamName string) error { 
	query := "insert into teams (team_name) values ($1);"
	_, err := r.db.Exec(query, teamName)
	return err
}

func (r *Repo) GetTeamMembers(teamName string) ([]entities.User, error) {
	query := "select id, username, team_name, is_active from users where team_name = $1 order by username;"
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	} 
	defer rows.Close()

	var users []entities.User
	for rows.Next() {
		var user entities.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}


func (r *Repo) GetTeam(teamName string) (*entities.Team, error) { //почему не по id? исправить
	var exists bool 
	err := r.db.QueryRow("select exists(select * from teams where team_name = $1);", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, sql.ErrNoRows
	}

	members, err := r.GetTeamMembers(teamName)
	if err != nil {
		return nil, err
	}

	teamMembers := make([]entities.TeamMember, len(members))
	for i, mem := range members {
		teamMembers[i] = entities.TeamMember{
			UserID: mem.ID,
			Username: mem.Username,
			IsActive: mem.IsActive,
		}
	}

	return &entities.Team{
		TeamName: teamName,
		Members: teamMembers,
	}, nil
}

func (r *Repo) TeamExists(teamName string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("select exists(select * from teams where team_name = $1);", teamName).Scan(&exists)
	return exists, err
}

func (r *Repo) CreateOrUpdateUser(user *entities.User) error {
	query := `insert into users (id, username, team_name, is_active, updated_at) 
				values ($1, $2, $3, $4, $5)
				on conflict (id)
				do update set
				username = excluded.username,
				team_name = excluded.team_name,
				is_active = excluded.is_active,
				updated_at = excluded.updated_at;`
	_, err := r.db.Exec(query, user.ID, user.Username, user.TeamName, user.IsActive, time.Now())
	return err
}

func (r *Repo) GetUser(id string) (*entities.User, error) {
	var user entities.User
	query := "select id, username, team_name, is_active from users where id = $1;"
	err := r.db.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		return nil, err
	}

	return &user, nil
}	

func (r *Repo) SetUserActive(id string, isActive bool) error {
	query := "update users set is_active = $1, updated_at = $2 where id = $3;"
	res, err := r.db.Exec(query, isActive, time.Now(), id)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err	
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repo) GetActiveTeamMembers(teamName string, excludeUser []string) ([]entities.User, error) {
	query := "select id, username, team_name, is_active from users where team_name = $1 and is_active = true"

	args := []interface{}{teamName}

	if len(excludeUser) > 0 {
		placeholders := ""
		for i, id := range excludeUser {
			if i > 0 {
				placeholders += ", "
			}
			placeholders += fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}
		query += fmt.Sprintf(" and id not in (%s)", placeholders)
	}
	query += ";"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entities.User
	for rows.Next() {
		var user entities.User
		if err := rows.Scan(&user.ID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, rows.Err()
}

func (r *Repo) CreatePullRequest(pr *entities.PullRequest, revIds []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "insert into pull_requests (id, name, author_id, status, created_at) values ($1, $2, $3, $4, $5);"
	_, err = tx.Exec(query, pr.ID, pr.Name, pr.AuthorID, pr.Status, time.Now())
	if err != nil {
		return err
	}

	for _, revId := range revIds {
		query = "insert into pr_reviewers (pull_request_id, user_id) values ($1, $2);"
		_, err = tx.Exec(query, pr.ID, revId)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repo) GetPullRequest(prID string) (*entities.PullRequest, error) {
	var pr entities.PullRequest

	query := "select id, name, author_id, status, created_at, merged_at from pull_requests where id = $1;"
	err := r.db.QueryRow(query, prID).Scan(
		&pr.ID,
		&pr.Name,
		&pr.AuthorID,
		&pr.Status,
		&pr.CreatedAt,
		&pr.MergedAt,
	)
	if err != nil {
		return nil, err
	}

	reviewers, err := r.GetPRReviewers(prID)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *Repo) GetPRReviewers(prID string) ([]string, error) {
	query := "select user_id from pr_reviewers where pull_request_id = $1 order by assigned_at;"
	rows, err := r.db.Query(query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, userID)
	}

	return reviewers, rows.Err()
}

func (r *Repo) PRExists(prID string) (bool, error) {
	var exists bool
	query := "select exists (select 1 from pull_requests where id = $1);"
	err := r.db.QueryRow(query, prID).Scan(&exists)
	return exists, err
}

func (r *Repo) UpdatePRStatus(prID string, status string, mergedAt *time.Time) error {
	query := "update pull_requests set status = $1, merged_at = $2 where id = $3;"
	result, err := r.db.Exec(query, status, mergedAt, prID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

func (r *Repo) RemoveReviewer(prID string, userID string) error {
	query := "delete from pr_reviewers where pull_request_id = $1 and user_id = $2;"
	result, err := r.db.Exec(query, prID, userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}

func (r *Repo) AddReviewer(prID string, userID string) error {
	query := "insert into pr_reviewers (pull_request_id, user_id) values ($1, $2);"
	_, err := r.db.Exec(query, prID, userID)
	return err
}

func (r *Repo) ReplaceReviewer(prID string, oldUserID string, newUserID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "delete from pr_reviewers where pull_request_id = $1 and user_id = $2;"
	_, err = tx.Exec(query, prID, oldUserID)
	if err != nil {
		return err
	}

	query = "insert into pr_reviewers (pull_request_id, user_id) values ($1, $2);"
	_, err = tx.Exec(query, prID, newUserID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repo) GetUserReviews(userID string) ([]entities.PullRequestShort, error) {
	query := `
		select pr.id, pr.name, pr.author_id, pr.status
		from pull_requests pr
		join pr_reviewers prr on pr.id = prr.pull_request_id
		where prr.user_id = $1
		order by pr.created_at desc;
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []entities.PullRequestShort
	for rows.Next() {
		var pr entities.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}



