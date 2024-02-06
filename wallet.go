package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	// "time"
)

type Wallet struct {
	Id      string  `json:"walletId"`
	Balance float64 `json:"balance"`
}

// TODO: figure out time
type Transaction struct {
	// Date   time.Time `json:"time"`
	FromId string  `json:"from"` // PK,FK
	ToId   string  `json:"to"`
	Amount float64 `json:"amount"`
}

// ---------
type WalletDoesNotExistsError struct{}

func (*WalletDoesNotExistsError) Error() string {
	return "Wallet's not found"
}

type WalletBalanceOverdraftError struct{}

func (*WalletBalanceOverdraftError) Error() string {
	return "Wallet's balance too low"
}

// ---------
type InMemoryDatabase struct {
	ids          map[string]Wallet
	transactions map[string][]Transaction
}

func (InMemoryDatabase) addWallet() (string, error) {
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

func (InMemoryDatabase) getWallet(id string) (Wallet, error) {
	if val, ok := db.ids[id]; ok {
		return val, nil
	}
	return Wallet{}, &WalletDoesNotExistsError{}
}

func (InMemoryDatabase) updateWallet(wallet Wallet) error {
	if _, ok := db.ids[wallet.Id]; ok {
		db.ids[wallet.Id] = wallet
		return nil
	}
	return &WalletDoesNotExistsError{}
}

func (InMemoryDatabase) createTransaction(tr Transaction) error {
	walletFrom, err := db.getWallet(tr.FromId)
	if err != nil {
		return err
	}
	walletTo, err := db.getWallet(tr.ToId)
	if err != nil {
		return err
	}
	if tr.Amount > walletFrom.Balance {
		return &WalletBalanceOverdraftError{}
	}
	walletFrom.Balance -= tr.Amount
	walletTo.Balance += tr.Amount

	// make transaction -- do this in an actual db transaction
	_ = db.updateWallet(walletFrom)
	_ = db.updateWallet(walletTo)
	// if _, ok := db.transactions[walletFrom.Id]; !ok {
	//     db.transactions[walletFrom.Id]= make([]Transaction, 0)
	// }
	db.transactions[walletFrom.Id] = append(db.transactions[walletFrom.Id], tr)
	return nil
}

func (InMemoryDatabase) getTransactions(walletId string) ([]Transaction, error) {
	_, err := db.getWallet(walletId)
	if err != nil {
		return nil, err
	}
	return db.transactions[walletId], nil
}

var db *InMemoryDatabase

// ---------

type WalletHandler struct{}

func NewWalletHandler() *WalletHandler {
	return &WalletHandler{}
}

func (h *WalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && WalletRe.MatchString(r.URL.Path):
		h.AddWallet(w, r)
	case r.Method == http.MethodPost && TransactionRe.MatchString(r.URL.Path):
		h.CreateTransaction(w, r)
	case r.Method == http.MethodGet && TransactionHistRe.MatchString(r.URL.Path):
		h.GetTransactionHistory(w, r)
	case r.Method == http.MethodGet && WalletStateRe.MatchString(r.URL.Path):
		h.GetWallet(w, r)
	default:
		return
	}
}

// must follow ID rules
var (
	WalletRe          = regexp.MustCompile(`^/api/v1/wallet$`)
	TransactionRe     = regexp.MustCompile(`^/api/v1/wallet/([a-zA-Z0-9]{4})/send$`)
	TransactionHistRe = regexp.MustCompile(`^/api/v1/wallet/([a-zA-Z0-9]{4})/history$`)
	WalletStateRe     = regexp.MustCompile(`^/api/v1/wallet/([a-zA-Z0-9]{4})$`)
)

func (h *WalletHandler) AddWallet(w http.ResponseWriter, r *http.Request) {
	walletId, err := db.addWallet()
	if err != nil {
		fmt.Fprintf(w, "%v\n", err)
		return
		// TODO: handle error
	}
	fmt.Fprintf(w, "%v\n", walletId)
}

func (h *WalletHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	wallet, err := getWalletFromPath(TransactionRe, r.URL.Path)
	if err != nil {
		// TODO 404
		fmt.Fprintf(w, "Error - 404, url wallet dne\n")
		return
	}
	var transaction Transaction
	err = json.NewDecoder(r.Body).Decode(&transaction)
	if err != nil {
		// Todo 400
		fmt.Fprintf(w, "Error - 400, bad body\n")
		return
	}
	if transaction.Amount < 0 {
		fmt.Fprintf(w, "Error - 400, bad body\n")
		return
	}
	if wallet.Id != transaction.FromId {
		fmt.Fprintf(w, "Error - 400, wallet url doesn't match wallet from\n")
		return
	}
	// TODO handle too poor error
	db.createTransaction(transaction)
	fmt.Fprintf(w, "Transaction succass\n")
}

func (h *WalletHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	matches := TransactionHistRe.FindStringSubmatch(r.URL.Path)
	hist, err := db.getTransactions(matches[1])
	if err != nil {
		// TODO 404
		fmt.Fprintf(w, "Error - 404, url wallet dne\n")
		return
	}
	fmt.Printf("%v\n", hist)
	jsonBytes, err := json.Marshal(hist)
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := getWalletFromPath(WalletStateRe, r.URL.Path)
	if err != nil {
		fmt.Fprintf(w, "Error - 404, url wallet dne\n")
		return
	}
	jsonBytes, err := json.Marshal(wallet)
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func getWalletFromPath(re *regexp.Regexp, path string) (Wallet, error) {
	matches := re.FindStringSubmatch(path)
	wallet, err := db.getWallet(matches[1])
	return wallet, err
}

// ---------

func main() {
	db = new(InMemoryDatabase)
	db.ids = map[string]Wallet{}
	db.transactions = map[string][]Transaction{}

	// add test values
	db.ids["abcd"] = Wallet{"abcd", 100.}
	db.ids["bcda"] = Wallet{"bcda", 20.}

	walletHandler := NewWalletHandler()
	mux := http.NewServeMux()
	mux.Handle("/api/v1/wallet", walletHandler)
	mux.Handle("/api/v1/wallet/", walletHandler)
	http.ListenAndServe(":8080", mux)
}
