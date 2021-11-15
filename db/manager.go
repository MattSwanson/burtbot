package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
)

var DbPool *pgxpool.Pool

// Connect will establish a connection to the pgql database based on the
// connection string stored in the env var PG_URL. Make sure to defer
// db.Close() to ensure proper closure of the pool
func Connect() (error, func()) {
	var err error
	DbPool, err = pgxpool.Connect(context.Background(), os.Getenv("PG_URL"))
	if err != nil {
		return err, nil
	}
	return nil, func() {
		if DbPool != nil {
			DbPool.Close()
		}
	}
}

func Check() {
	rows, err := DbPool.Query(context.Background(), "SELECT * FROM users")
	if err != nil {
		fmt.Println("couldn't get users")
		return
	}
	defer rows.Close()
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			fmt.Println("error getting row values")
		}
		fmt.Println(vals)
	}
}
