package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type SetRequest struct {
	Text   string `json:"text"`
	Device string `json:"device"`
}

//go:embed index.html
var indexHTML []byte

var (
	clipboard       string
	mu              sync.RWMutex
	clearTimer      *time.Timer
	timerGeneration int64
)

func clipboardHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		defer mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(clipboard))

	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		var req SetRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		mu.Lock()
		defer mu.Unlock()
		clipboard = req.Text

		if clearTimer != nil {
			clearTimer.Stop()
		}

		timerGeneration++
		currentGen := timerGeneration
		clearTimer = time.AfterFunc(60*time.Second, func() {
			mu.Lock()
			defer mu.Unlock()
			if timerGeneration == currentGen {
				clipboard = ""
			}
		})

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := host + ":" + port

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/clipboard", clipboardHandler)

	fmt.Printf("Clipshare server starting on http://%s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
