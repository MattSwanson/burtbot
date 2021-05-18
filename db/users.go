package db

import (
	"errors"
	"context"
)

type User struct {
	TwitchID int
	DisplayName string
}

func GetUser(twitchID int) (User, error) {
	user := User{}
	if DbPool == nil {
		return user, errors.New("no connection to db present")
	}
	err := DbPool.QueryRow(context.Background(), "SELECT * FROM users WHERE twitch_id=$1", twitchID).Scan(&user.TwitchID, &user.DisplayName)
	if err != nil {
		return user, err
	}
	return user, nil
}

func GetUserByName(userName string) (User, error) {
	user := User{}
	err := DbPool.QueryRow(context.Background(), "SELECT * FROM users WHERE display_name='$1'", userName).Scan(&user.TwitchID, &user.DisplayName)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func GetUsers() ([]User, error) {
	users := []User{}
	rows, err := DbPool.Query(context.Background(), "SELECT * FROM users")
	if err != nil {
		return users, err
	}
	defer rows.Close()
	for rows.Next() {
		user := User{}
		err := rows.Scan(&user.TwitchID, &user.DisplayName)
		if err != nil {
			return []User{}, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return []User{}, err
	}	
	return users, nil
}

func AddUser(user User) error {
	_, err := DbPool.Exec(context.Background(), "INSERT INTO users (twitch_id, display_name) VALUES ($1, $2)", user.TwitchID, user.DisplayName)
	if err != nil {
		return err
	}
	return nil
}
