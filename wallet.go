package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"time"
)

type Wallet struct {
	Id      string  `json:"walletId"`
	Balance float64 `json:"balance"`
}

// TODO: figure out time
type Transaction struct {
	Time   string `json:"time"`
	FromId string  `json:"from"` // PK,FK
	ToId   string  `json:"to"`
	Amount float64 `json:"amount"`
}


// ---------
type BadRequestError struct {}

func (*BadRequestError) Error() string {
    return "Ошибка в запросе"
}

type WalletDNEError struct{}

func (*WalletDNEError) Error() string {
	return "Указанный кошелёк не найден"
}

type FromWalletDNEError struct{}

func (*FromWalletDNEError) Error() string {
    return "Исходящий кошелек не найден"
}

type BadRequestOverdraftError struct{}

func (*BadRequestOverdraftError) Error() string {
	return "Ошибка в пользовательском запросе или ошибка перевода"
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
	return Wallet{}, &WalletDNEError{}
}

func (InMemoryDatabase) updateWallet(wallet Wallet) error {
	if _, ok := db.ids[wallet.Id]; ok {
		db.ids[wallet.Id] = wallet
		return nil
	}
	return &WalletDNEError{}
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
		return &BadRequestOverdraftError{}
	}
	walletFrom.Balance -= tr.Amount
	walletTo.Balance += tr.Amount

	// make transaction -- do this in an actual db transaction
	_ = db.updateWallet(walletFrom)
	_ = db.updateWallet(walletTo)
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
        handleError(w, http.StatusServiceUnavailable, "")
        fmt.Println("AddWalllet")
	}
	fmt.Fprintf(w, "%v\n", walletId)
}

func (h *WalletHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	wallet, err := getWalletFromPath(TransactionRe, r.URL.Path)
	if err != nil {
        err = &FromWalletDNEError{}
        handleError(w, http.StatusNotFound, err.Error())
        fmt.Println("CreateTransaction - wallet dne")
		return
	}
	var transaction Transaction
	err = json.NewDecoder(r.Body).Decode(&transaction)
    _, timeErr := time.Parse(time.RFC3339, transaction.Time)
	if err != nil || timeErr != nil || transaction.Amount < 0 || wallet.Id != transaction.FromId {
        err = &BadRequestOverdraftError{}
        handleError(w, http.StatusBadRequest, err.Error())
        fmt.Println("CreateTransaction - transaction format error")
		return
	}
	err = db.createTransaction(transaction)
    if err != nil {
        err = &BadRequestOverdraftError{}
        handleError(w, http.StatusBadRequest, err.Error())
        fmt.Printf("CreateTransaction - couldn't create a transaction %v\n", transaction)
    }
    w.WriteHeader(http.StatusOK)
}

func (h *WalletHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	matches := TransactionHistRe.FindStringSubmatch(r.URL.Path)
	hist, err := db.getTransactions(matches[1])
	if err != nil {
        err = &WalletDNEError{}
        handleError(w, http.StatusNotFound, err.Error())
        fmt.Println("GetTransactionHistory")
		return
	}
	jsonBytes, err := json.Marshal(hist)
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	wallet, err := getWalletFromPath(WalletStateRe, r.URL.Path)
	if err != nil {
        err = &WalletDNEError{}
        handleError(w, http.StatusNotFound, err.Error())
        fmt.Println("GetWallet")
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

func handleError(w http.ResponseWriter, status int, msg string) {
    w.WriteHeader(status)
    jsonMsg, err := json.Marshal(msg)
    if err != nil {
        return
    }
    w.Write(jsonMsg)
}

// ---------

func main() {
	db = new(InMemoryDatabase)
	db.ids = map[string]Wallet{}
	db.transactions = map[string][]Transaction{}

	// add test values
	db.ids["abcd"] = Wallet{"abcd", 100.}
	db.ids["bcda"] = Wallet{"bcda", 20.}

	walletHandler := new(WalletHandler)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/wallet", walletHandler)
	mux.Handle("/api/v1/wallet/", walletHandler)
	http.ListenAndServe(":8080", mux)
}
