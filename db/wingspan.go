package db

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type WingspanBird struct {
	ID             int
	CommonName     string
	ScientificName string
	Set            string
	Color          string
	PowerText      string
	FlavorText     string
	IsPredator     bool
	IsFlocking     bool
	IsBonusCard    bool
	VictoryPoints  int
	NestType       string
	EggLimit       int
	Wingspan       int
	HasPlayed      bool
	Forest         bool
	Grassland      bool
	Wetland        bool
}

func LoadWingspanDataFromCSV() {
	f, err := os.ReadFile("fixed-wingspan-birds.csv")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(f))
	scanner.Scan()
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ",")
		points, _ := strconv.ParseInt(fields[9], 10, 64)
		eggLimit, _ := strconv.ParseInt(fields[11], 10, 64)
		wingspan, err := strconv.ParseInt(fields[12], 10, 64)
		if err != nil {
			wingspan = 0
		}
		bird := WingspanBird{
			CommonName:     fields[0],
			ScientificName: fields[1],
			Set:            fields[2],
			Color:          fields[3],
			PowerText:      strings.ReplaceAll(fields[4], "|", ","),
			FlavorText:     strings.ReplaceAll(fields[5], "|", ","),
			IsPredator:     fields[6] != "",
			IsFlocking:     fields[7] != "",
			IsBonusCard:    fields[8] != "",
			VictoryPoints:  int(points),
			NestType:       fields[10],
			EggLimit:       int(eggLimit),
			Wingspan:       int(wingspan),
			Forest:         fields[13] != "",
			Grassland:      fields[14] != "",
			Wetland:        fields[15] != "",
		}
		if err := AddWingspanBird(bird); err != nil {
			fmt.Println(err)
		}
	}
}

func AddWingspanBird(b WingspanBird) error {
	_, err := DbPool.Exec(context.Background(),
		`INSERT INTO wingspan_birds (common_name, scientific_name, set, color, power_text, 
			flavor_text, predator, flocking, bonus_card, victory_points, nest_type, egg_limit, 
			wingspan, forest, grassland, wetland)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		b.CommonName, b.ScientificName, b.Set, b.Color, b.PowerText, b.FlavorText, b.IsPredator, b.IsFlocking, b.IsBonusCard,
		b.VictoryPoints, b.NestType, b.EggLimit, b.Wingspan, b.Forest, b.Grassland, b.Wetland)

	return err
}

func MarkBirdPlayed(birdID int, isPlayed bool) error {
	_, err := DbPool.Exec(context.Background(),
		`UPDATE wingspan_birds
		SET has_played = $1
		WHERE id = $2`, isPlayed, birdID)
	return err
}

func GetUnplayedBirds() ([]WingspanBird, error) {
	birds := []WingspanBird{}
	rows, err := DbPool.Query(context.Background(),
		`SELECT * FROM wingspan_birds
		 WHERE has_played = false
		 AND set != 'asia'
		 ORDER BY common_name ASC`)
	if err != nil {
		return birds, err
	}
	defer rows.Close()
	for rows.Next() {
		b := WingspanBird{}
		err := rows.Scan(
			&b.ID, &b.CommonName, &b.ScientificName, &b.Set, &b.Color, &b.PowerText, &b.FlavorText,
			&b.IsPredator, &b.IsFlocking, &b.IsBonusCard, &b.VictoryPoints, &b.NestType, &b.EggLimit,
			&b.Wingspan, &b.HasPlayed, &b.Forest, &b.Grassland, &b.Wetland)
		if err != nil {
			return []WingspanBird{}, err
		}
		birds = append(birds, b)
	}
	err = rows.Err()
	return birds, err

}

func GetUnplayedBirdCount() (int, error) {
	unplayed := 0
	row := DbPool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM wingspan_birds WHERE has_played = false;`)
	err := row.Scan(&unplayed)
	if err != nil {
		return 0, err
	}
	return unplayed, nil
}

func GetUnplayedBirdCountBySet() (map[string]int, error) {
	results := make(map[string]int)
	rows, err := DbPool.Query(context.Background(),
		`SELECT set, SUM(1) FROM wingspan_birds
		 WHERE has_played = false AND set != 'asia'
		 GROUP BY set`)
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		var set string
		var count int
		err := rows.Scan(&set, &count)
		if err != nil {
			return make(map[string]int), err
		}
		results[set] = count
	}
	err = rows.Err()
	return results, err
}
