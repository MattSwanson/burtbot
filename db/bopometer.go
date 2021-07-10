package db

import (
	"context"
	"time"
)

type BopRating struct {
	SpotifyID string
	SongName string
	SongArtists string
	Rating float32
	AddedAt time.Time
}

// Doesn't return an insert ID since the song id will be
// the unique key
func AddBopRating(b BopRating) (error) {
	_, err := DbPool.Exec(context.Background(),
	`INSERT INTO bopometer_ratings (spotify_id, song_name, song_artists, rating, added_at)
	 VALUES ($1, $2, $3, $4, $5)`, b.SpotifyID, b.SongName, b.SongArtists, b.Rating, b.AddedAt)
	return err
}

func GetBopRatings(limit int) ([]BopRating, error) {
	ratings := []BopRating{}
	rows, err := DbPool.Query(context.Background(),
	`SELECT * FROM bopometer_ratings
	 ORDER BY rating DESC
	 LIMIT $1`, limit)
	if err != nil {
		return ratings, err
	}
	defer rows.Close()
	for rows.Next() {
		r := BopRating{}
		err := rows.Scan(&r.SpotifyID, &r.SongName, &r.SongArtists, &r.Rating, &r.AddedAt)
		if err != nil {
			return []BopRating{}, err
		}
		ratings = append(ratings, r)
	}
	if err = rows.Err(); err != nil {
		return ratings, err
	}
	return ratings, nil
}

func GetBopRating(spotifyID string) (BopRating, error) {
	row := DbPool.QueryRow(context.Background(),
	`SELECT * FROM bopometer_ratings
	 WHERE spotify_id = $1`, spotifyID)
	br := BopRating{}
	err := row.Scan(&br.SpotifyID, &br.SongName, &br.SongArtists,
					&br.Rating, &br.AddedAt)
	if err != nil {
		return BopRating{}, err
	}
	return br, nil
}

func UpdateBopRating(trackID string, rating float32) error {
	_, err := DbPool.Exec(context.Background(),
	`UPDATE bopometer_ratings 
	 SET rating = $1, added_at = $2
	 WHERE spotify_id = $3`, rating, time.Now(), trackID)
	return err
}
