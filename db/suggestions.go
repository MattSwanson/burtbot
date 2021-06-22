package db

import (
	"context"
	"time"
)

type Suggestion struct {
	Username string
	UserID	 int
	Date     time.Time
	Text     string
}

func AddSuggestion(s Suggestion) error {

	// check to see if this twitch user is present
	// in the users table
	u, _ := GetUser(s.UserID)
	if u.TwitchID == 0 {
		err := AddUser(User{s.UserID, s.Username})
		if err != nil {
			return err
		}
	}	
	_, err := DbPool.Exec(context.Background(),
	`INSERT INTO suggestions (suggestion, user_id, submitted_on)
	 VALUES ($1, $2, $3)`, s.Text, s.UserID, s.Date)
	if err != nil {
		return err
	}
	return nil
}

func GetSuggestions() ([]Suggestion, error) {
	suggestions := []Suggestion{}
	rows, err := DbPool.Query(context.Background(), 
		`SELECT 
		  suggestions.user_id,
		  users.display_name,
		  suggestions.suggestion,
		  suggestions.submitted_on
		 FROM suggestions
		 JOIN users ON users.twitch_id = suggestions.user_id
		`)
	if err != nil {
		return suggestions, err
	}
	defer rows.Close()
	for rows.Next() {
		s := Suggestion{}
		err := rows.Scan(&s.UserID, &s.Username, &s.Text, &s.Date)
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
