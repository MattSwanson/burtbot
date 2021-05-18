package db

import (
	"context"
)

type Wallet struct {
	Balances []Balance
}

type Balance struct {
	UserID		int // user id
	CurrencyID	int // currency id
	Balance		float32 // currency balance
}

// Get Wallet - get a users wallet. All balances related to that user
func GetWallet(userID int) (*Wallet, error) {
	query := `
		SELECT * FROM wallets
		WHERE user_id = $1
	`
	rows, err := DbPool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	wallet := Wallet{[]Balance{}}
	defer rows.Close()
	for rows.Next() {
		b := Balance{}
		err := rows.Scan(&b.UserID, &b.CurrencyID, &b.Balance)
		if err != nil {
			return nil, err
		}
		wallet.Balances = append(wallet.Balances, b)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &wallet, nil
}

// Update Balance - update a balance in a users wallet
// should we just take in one Balance to update??
func UpdateBalance(userID, currencyID int) (Balance, error) {
	// return the updated balance row
	return Balance{}, nil
}
