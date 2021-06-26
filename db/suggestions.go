package db

import (
	"context"
	"time"
	"sync"
)

var suggestionLock sync.Mutex = sync.Mutex{}

type Suggestion struct {
	ID		 int
	Username string
	UserID	 int
	Date     time.Time
	Text     string
	Complete bool
}

func AddSuggestion(s Suggestion) (int, error) {

	// check to see if this twitch user is present
	// in the users table
	u, _ := GetUser(s.UserID)
	if u.TwitchID == 0 {
		err := AddUser(User{s.UserID, s.Username})
		if err != nil {
			return 0, err
		}
	}	
	suggestionLock.Lock()
	defer suggestionLock.Unlock()
	_, err := DbPool.Exec(context.Background(),
	`INSERT INTO suggestions (suggestion, user_id, submitted_on)
	 VALUES ($1, $2, $3)`, s.Text, s.UserID, s.Date)
	if err != nil {
		return 0, err
	}
	row, err := DbPool.Query(context.Background(),
		`SELECT ID FROM suggestions
		 ORDER BY ID desc
		 LIMIT 1`)
	if err != nil {
		return 0, err 
	}
	defer row.Close()
	id := 0
	for row.Next() {
		err = row.Scan(&id)
		if err != nil {
			return id, err
		}
	}
	return id, nil
}

func GetSuggestions() ([]Suggestion, error) {
	suggestions := []Suggestion{}
	rows, err := DbPool.Query(context.Background(), 
		`SELECT
		  suggestions.id,
		  suggestions.user_id,
		  users.display_name,
		  suggestions.suggestion,
		  suggestions.submitted_on,
		  suggestions.complete
		 FROM suggestions
		 JOIN users ON users.twitch_id = suggestions.user_id
		 ORDER BY suggestions.id
		 `)
	if err != nil {
		return suggestions, err
	}
	defer rows.Close()
	for rows.Next() {
		s := Suggestion{}
		err := rows.Scan(&s.ID, &s.UserID, &s.Username, &s.Text, &s.Date, &s.Complete)
		if err != nil {
			return []Suggestion{}, err
		}
		suggestions = append(suggestions, s)
	}
	if err = rows.Err(); err != nil {
		return suggestions, err
	}
	return suggestions, nil
}

func DeleteSuggestion(id int) error {
	_, err := DbPool.Exec(context.Background(),
	`DELETE FROM suggestions 
	 WHERE id = $1`,
	 id)
	return err
}

func SetSuggestionCompletion(id int, b bool) error {
	_, err := DbPool.Exec(context.Background(),
	`UPDATE suggestions
	 SET complete = $1
	 WHERE id = $2`,
	b, id)
	return err
}
