package main

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/jackc/pgx/v5"
	// "github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	AddWallet(ctx context.Context) (string, error)
	GetWallet(ctx context.Context, id string) (Wallet, error)
	UpdateWallet(ctx context.Context, wallet Wallet) error
	CreateTransaction(ctx context.Context, tr Transaction) error
	GetTransactions(ctx context.Context, walletId string) ([]Transaction, error)
}

type PostgreSQLDb struct {
	Postgres *pgx.Conn
}

func (db *PostgreSQLDb) AddWallet(ctx context.Context) (string, error) {
	// generate random id
	var id string
	var err error
	for {
		id, err = randSeq(4)
		if err != nil {
			return "", err
		}
		_, err := db.GetWallet(ctx, id)
		if err == pgx.ErrNoRows {
			break
		}
	}
	_, err = db.Postgres.Exec(ctx, "INSERT INTO wallets (id, balance) VALUES ($1, $2)", id, 100)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (db *PostgreSQLDb) GetWallet(ctx context.Context, id string) (Wallet, error) {
	var wallet Wallet
	err := db.Postgres.QueryRow(ctx, "SELECT id, balance FROM wallets WHERE id = $1", id).Scan(&wallet.Id, &wallet.Balance)
	if err != nil {
		return Wallet{}, err
	}
	return wallet, nil
}

func (db *PostgreSQLDb) UpdateWallet(ctx context.Context, wallet Wallet) error {
	_, err := db.Postgres.Exec(ctx, "UPDATE wallets SET balance = $1 WHERE id = $2", wallet.Balance, wallet.Id)
	return err
}

func (db *PostgreSQLDb) CreateTransaction(ctx context.Context, tr Transaction) error {
	// validate transactions the same way as in InMemoryDatabase
	walletFrom, err := db.GetWallet(ctx, tr.FromId)
	if err != nil {
		return err
	}
	walletTo, err := db.GetWallet(ctx, tr.ToId)
	if err != nil {
		return err
	}
	if tr.Amount > walletFrom.Balance {
		return &BadRequestOverdraftError{}
	}
	walletFrom.Balance -= tr.Amount
	walletTo.Balance += tr.Amount
	_ = db.UpdateWallet(ctx, walletFrom)
	_ = db.UpdateWallet(ctx, walletTo)
	_, err = db.Postgres.Exec(ctx, "INSERT INTO transactions (time, from_id, to_id, amount) VALUES ($1::timestamptz, $2, $3, $4)", tr.TimeRfc, tr.FromId, tr.ToId, tr.Amount)
	return err
}

func (db *PostgreSQLDb) GetTransactions(ctx context.Context, walletId string) ([]Transaction, error) {
	var transactions []Transaction
	rows, err := db.Postgres.Query(ctx, "SELECT time, from_id, to_id, amount FROM transactions WHERE from_id = $1 OR to_id = $1", walletId)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tr Transaction
		err = rows.Scan(&tr.TimeRfc, &tr.FromId, &tr.ToId, &tr.Amount)
		if err != nil {
			return nil, err
		}
        tr.Time = tr.TimeRfc.Format("2006-01-02 15:04:05")
		transactions = append(transactions, tr)
	}
	return transactions, nil
}

// ---------
type InMemoryDatabase struct {
	ids          map[string]Wallet
	transactions map[string][]Transaction
}

func (db *InMemoryDatabase) AddWallet(_ context.Context) (string, error) {
	for {
		id, err := randSeq(4)
		if err != nil {
			return "", err
		}
		if _, ok := db.ids[id]; ok {
			continue
		}
		db.ids[id] = Wallet{
			Id:      id,
			Balance: 100.,
		}
		return id, nil
	}
}

// https://stackoverflow.com/a/75597742/9817178
func randSeq(n int) (string, error) {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	b := make([]rune, n)
	for i := 0; i < n; i++ {
		r, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[r.Int64()]
	}
	return string(b), nil
}

func (db *InMemoryDatabase) GetWallet(_ context.Context, id string) (Wallet, error) {
	if val, ok := db.ids[id]; ok {
		return val, nil
	}
	return Wallet{}, &WalletDNEError{}
}

func (db *InMemoryDatabase) UpdateWallet(_ context.Context, wallet Wallet) error {
	if _, ok := db.ids[wallet.Id]; ok {
		db.ids[wallet.Id] = wallet
		return nil
	}
	return &WalletDNEError{}
}

func (db *InMemoryDatabase) CreateTransaction(ctx context.Context, tr Transaction) error {
	walletFrom, err := db.GetWallet(ctx, tr.FromId)
	if err != nil {
		return err
	}
	walletTo, err := db.GetWallet(ctx, tr.ToId)
	if err != nil {
		return err
	}
	if tr.Amount > walletFrom.Balance {
		return &BadRequestOverdraftError{}
	}
	walletFrom.Balance -= tr.Amount
	walletTo.Balance += tr.Amount

	// make transaction -- do this in an actual db transaction
	_ = db.UpdateWallet(ctx, walletFrom)
	_ = db.UpdateWallet(ctx, walletTo)
	db.transactions[walletFrom.Id] = append(db.transactions[walletFrom.Id], tr)
	return nil
}

func (db *InMemoryDatabase) GetTransactions(ctx context.Context, walletId string) ([]Transaction, error) {
	_, err := db.GetWallet(ctx, walletId)
	if err != nil {
		return nil, err
	}
	return db.transactions[walletId], nil
}
