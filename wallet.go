package main

import (
	"context"
	"encoding/json"
	"fmt"
    "os"
	"net/http"
	"regexp"
	"time"
    "github.com/jackc/pgx/v5"
)

type Wallet struct {
	Id      string  `json:"walletId"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	Time   string `json:"time"`
	FromId string  `json:"from"` // PK,FK
	ToId   string  `json:"to"`
	Amount float64 `json:"amount"`
    TimeRfc time.Time
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
	walletId, err := db.AddWallet(context.Background())
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
	var tr Transaction
	err = json.NewDecoder(r.Body).Decode(&tr)
    timeRfc, timeErr := time.Parse(time.RFC3339, tr.Time)
    tr.TimeRfc = timeRfc
	if err != nil || timeErr != nil || tr.Amount < 0 || wallet.Id != tr.FromId {
        err = &BadRequestOverdraftError{}
        handleError(w, http.StatusBadRequest, err.Error())
        fmt.Println("CreateTransaction - transaction format error")
		return
	}
	err = db.CreateTransaction(context.Background(), tr)
    if err != nil {
        // err = &BadRequestOverdraftError{}
        handleError(w, http.StatusBadRequest, err.Error())
        fmt.Printf("CreateTransaction - couldn't create a transaction %v\n", tr)
    }
}

func (h *WalletHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	matches := TransactionHistRe.FindStringSubmatch(r.URL.Path)
	hist, err := db.GetTransactions(context.Background(), matches[1])
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
	wallet, err := db.GetWallet(context.Background(), matches[1])
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

var db *PostgreSQLDb
func main() {
    conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL"))
    if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

    db = new(PostgreSQLDb)
    db.Postgres = conn

	walletHandler := new(WalletHandler)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/wallet", walletHandler)
	mux.Handle("/api/v1/wallet/", walletHandler)
	http.ListenAndServe(":8080", mux)
}
