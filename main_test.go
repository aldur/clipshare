package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
		body           interface{}
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
	
	const numGoroutines = 10
	const testText = "concurrent test"
	
	setReq := SetRequest{Text: testText, Device: "test"}
	body, _ := json.Marshal(setReq)
	
	req := httptest.NewRequest(http.MethodPost, "/clipboard", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	clipboardHandler(w, req)
	
	done := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
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
	
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}