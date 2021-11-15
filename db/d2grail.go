package db

import (
	"context"
	"time"
)

type GrailItem struct {
	ID        int
	Name      string
	SetName   string
	BaseItem  string
	Found     time.Time
	BaseLevel int
	Rarity    int
}

func AddGrailItem(i GrailItem) error {
	_, err := DbPool.Exec(context.Background(),
		`INSERT INTO d2_grail_items (name, set_name, base_item, base_level)
	 VALUES ($1, $2, $3, $4)`, i.Name, i.SetName, i.BaseItem, i.BaseLevel)
	return err
}

func GetUnfoundGrailItems() ([]GrailItem, error) {
	items := []GrailItem{}
	rows, err := DbPool.Query(context.Background(),
		`SELECT * FROM d2_grail_items
		 WHERE found < '1000-01-01'
		 ORDER BY base_item`)
	if err != nil {
		return items, err
	}
	defer rows.Close()
	for rows.Next() {
		i := GrailItem{}
		err := rows.Scan(&i.ID, &i.Name, &i.SetName, &i.BaseItem, &i.Found, &i.BaseLevel)
		if err != nil {
			return []GrailItem{}, err
		}
		items = append(items, i)
	}
	err = rows.Err()
	return items, err
}

func GetLastFoundItems(limit int) ([]GrailItem, error) {
	items := []GrailItem{}
	rows, err := DbPool.Query(context.Background(),
		`SELECT * FROM d2_grail_items
		 WHERE found > '1000-01-01'
		 ORDER BY found desc
		 LIMIT $1`, limit)
	if err != nil {
		return items, err
	}
	defer rows.Close()
	for rows.Next() {
		i := GrailItem{}
		err := rows.Scan(&i.ID, &i.Name, &i.SetName, &i.BaseItem, &i.Found, &i.BaseLevel)
		if err != nil {
			return []GrailItem{}, err
		}
		items = append(items, i)
	}
	err = rows.Err()
	return items, err
}

// MarkItemFound updates the 'found' status of the item
// A zero value time can be given to make it 'unfound'
// Otherwise the given timestamp will mark when in was
// found
func MarkItemFound(itemID int, t time.Time) error {
	_, err := DbPool.Exec(context.Background(),
		`UPDATE d2_grail_items
		 SET found = $1
		 WHERE item_id = $2`, t, itemID)
	return err
}
