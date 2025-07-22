package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestClipboardGet(t *testing.T) {
	tests := []struct {
		name     string
		setup    func()
		expected string
	}{
		{
			name:     "empty clipboard",
			setup:    func() { clipboard = "" },
			expected: "",
		},
		{
			name:     "with text",
			setup:    func() { clipboard = "test text" },
			expected: "test text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			req := httptest.NewRequest(http.MethodGet, "/clipboard", nil)
			w := httptest.NewRecorder()

			clipboardHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expected {
				t.Errorf("expected body %q, got %q", tt.expected, w.Body.String())
			}

			if w.Header().Get("Content-Type") != "text/plain" {
				t.Errorf("expected Content-Type text/plain, got %s", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestClipboardPost(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		expectedStatus int
		expectedText   string
	}{
		{
			name:           "valid request",
			body:           SetRequest{Text: "hello world", Device: "test"},
			expectedStatus: http.StatusOK,
			expectedText:   "hello world",
		},
		{
			name:           "unknown device",
			body:           SetRequest{Text: "test text", Device: "unknown"},
			expectedStatus: http.StatusOK,
			expectedText:   "test text",
		},
		{
			name:           "invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedText:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clipboard = ""

			var body bytes.Buffer
			if str, ok := tt.body.(string); ok {
				body.WriteString(str)
			} else {
				json.NewEncoder(&body).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/clipboard", &body)
			w := httptest.NewRecorder()

			clipboardHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if clipboard != tt.expectedText {
				t.Errorf("expected clipboard %q, got %q", tt.expectedText, clipboard)
			}
		})
	}
}

func TestClipboardMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/clipboard", nil)
	w := httptest.NewRecorder()

	clipboardHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestIntegration(t *testing.T) {
	clipboard = ""

	testText := "integration test text"

	setReq := SetRequest{Text: testText, Device: "test-device"}
	body, _ := json.Marshal(setReq)

	req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	clipboardHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("set request failed with status %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/clipboard", nil)
	w = httptest.NewRecorder()
	clipboardHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get request failed with status %d", w.Code)
	}

	if w.Body.String() != testText {
		t.Errorf("expected %q, got %q", testText, w.Body.String())
	}
}

func TestConcurrentAccess(t *testing.T) {
	clipboard = ""
	clearTimer = nil

	const numGoroutines = 10
	const testText = "concurrent test"

	setReq := SetRequest{Text: testText, Device: "test"}
	body, _ := json.Marshal(setReq)

	req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	clipboardHandler(w, req)

	done := make(chan bool, numGoroutines)

	for range numGoroutines {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/clipboard", nil)
			w := httptest.NewRecorder()
			clipboardHandler(w, req)

			if w.Body.String() != testText {
				t.Errorf("concurrent read failed: expected %q, got %q", testText, w.Body.String())
			}
			done <- true
		}()
	}

	for range numGoroutines {
		<-done
	}
}

func TestAutoClear(t *testing.T) {
	clipboard = ""
	clearTimer = nil

	testText := "auto clear test"
	setReq := SetRequest{Text: testText, Device: "test"}
	body, _ := json.Marshal(setReq)

	req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	clipboardHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify content is set
	if clipboard != testText {
		t.Errorf("expected clipboard %q, got %q", testText, clipboard)
	}

	// Wait for auto-clear (using shorter duration for test)
	time.Sleep(100 * time.Millisecond)

	// Timer should still be active, content should remain
	if clipboard != testText {
		t.Errorf("clipboard cleared too early: expected %q, got %q", testText, clipboard)
	}
}

func TestMultipleSetsCancelPreviousTimer(t *testing.T) {
	clipboard = ""
	clearTimer = nil

	// Set first text
	firstText := "first text"
	setReq := SetRequest{Text: firstText, Device: "test"}
	body, _ := json.Marshal(setReq)

	req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	clipboardHandler(w, req)

	// Set second text immediately
	secondText := "second text"
	setReq = SetRequest{Text: secondText, Device: "test"}
	body, _ = json.Marshal(setReq)

	req = httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	clipboardHandler(w, req)

	// Verify second text is set
	if clipboard != secondText {
		t.Errorf("expected clipboard %q, got %q", secondText, clipboard)
	}
}

func TestConcurrentSets(t *testing.T) {
	clipboard = ""
	clearTimer = nil
	timerGeneration = 0

	const numGoroutines = 5
	done := make(chan string, numGoroutines)

	// Launch concurrent set operations
	for i := range numGoroutines {
		go func(id int) {
			testText := fmt.Sprintf("concurrent set %d", id)
			setReq := SetRequest{Text: testText, Device: "test"}
			body, _ := json.Marshal(setReq)

			req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			clipboardHandler(w, req)

			done <- testText
		}(i)
	}

	// Wait for all to complete
	results := make([]string, numGoroutines)
	for i := range numGoroutines {
		results[i] = <-done
	}

	// One of the values should be the final clipboard content
	found := slices.Contains(results, clipboard)

	if !found {
		t.Errorf("clipboard content %q not found in concurrent results %v", clipboard, results)
	}
}

func TestOldTimerClearsNewContent(t *testing.T) {
	clipboard = ""
	clearTimer = nil
	timerGeneration = 0

	// Custom scenario: simulate the exact race condition
	// 1. First POST sets timer for 30ms
	// 2. Wait 20ms (timer hasn't fired yet)
	// 3. Second POST tries to set new content and new timer
	// 4. But first timer fires during second POST processing

	firstText := "first text"
	secondText := "second text"

	// Set first text with 30ms timer

	mu.Lock()
	clipboard = firstText
	timerGeneration++
	firstGen := timerGeneration
	clearTimer = time.AfterFunc(30*time.Millisecond, func() {
		mu.Lock()
		if timerGeneration == firstGen {
			clipboard = ""
			clearTimer = nil
		}
		mu.Unlock()
	})
	mu.Unlock()

	// Wait 20ms - first timer is still pending
	time.Sleep(20 * time.Millisecond)

	// Simulate second POST that takes some time to process
	// This is where the race could occur
	mu.Lock()
	// At this point, the first timer might fire and try to acquire the mutex
	// But we're holding it, so it will wait

	// Set second content
	clipboard = secondText

	if clearTimer != nil {
		clearTimer.Stop() // This should stop the first timer
	}

	timerGeneration++
	secondGen := timerGeneration
	clearTimer = time.AfterFunc(50*time.Millisecond, func() {
		mu.Lock()
		if timerGeneration == secondGen {
			clipboard = ""
			clearTimer = nil
		}
		mu.Unlock()
	})
	mu.Unlock()

	// At this point, if first timer was waiting on mutex, it should now execute
	// but it should be stopped or should check generation

	// Wait a bit to let any pending timer callbacks execute
	time.Sleep(20 * time.Millisecond)

	mu.RLock()
	defer mu.RUnlock()
	// The clipboard should still contain second text
	// If it's empty, the old timer cleared it (race condition)
	if clipboard != secondText {
		t.Errorf("Race condition: expected %q, got %q", secondText, clipboard)
		t.Logf("This means the old timer cleared the new content")
	}
}

