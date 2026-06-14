// exchange.go - Курсы валют на Go (веб-сервер + HTML-шаблон)
// Атрибуция: Данные предоставлены Exchange Rate API
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type ExchangeResponse struct {
	Result     string             `json:"result"`
	BaseCode   string             `json:"base_code"`
	Rates      map[string]float64 `json:"rates"`
	TimeUpdate string             `json:"time_last_update_utc"`
}

var (
	cachedRates  map[string]float64
	cacheMutex   sync.RWMutex
	lastUpdate   time.Time
	baseCurrency = "USD"
)

const apiURL = "https://open.er-api.com/v6/latest/"

func fetchRates(base string) (map[string]float64, error) {
	url := apiURL + base
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data ExchangeResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if data.Result != "success" {
		return nil, fmt.Errorf("API error: %s", data.Result)
	}
	return data.Rates, nil
}

func getRates(base string) (map[string]float64, error) {
	cacheMutex.RLock()
	if time.Since(lastUpdate) < time.Hour && cachedRates != nil {
		cacheMutex.RUnlock()
		return cachedRates, nil
	}
	cacheMutex.RUnlock()
	
	rates, err := fetchRates(base)
	if err != nil {
		return nil, err
	}
	cacheMutex.Lock()
	cachedRates = rates
	lastUpdate = time.Now()
	baseCurrency = base
	cacheMutex.Unlock()
	return rates, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("exchange.html"))
	tmpl.Execute(w, nil)
}

func apiRatesHandler(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	if base == "" {
		base = "USD"
	}
	rates, err := getRates(base)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rates)
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/rates", apiRatesHandler)
	
	fmt.Println("🚀 Сервер запущен на http://localhost:8080")
	fmt.Println("Атрибуция: Данные предоставлены Exchange Rate API")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
