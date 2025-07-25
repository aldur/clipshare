package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const testPort = "18080"
const testHost = "localhost"
const testURL = "http://" + testHost + ":" + testPort

func startTestServer(t *testing.T) (context.CancelFunc, error) {
	t.Helper()
	
	// Set environment variables for the test server
	os.Setenv("HOST", testHost)
	os.Setenv("PORT", testPort)
	
	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "go", "run", "main.go")
	cmd.Env = append(os.Environ(), "HOST="+testHost, "PORT="+testPort)
	
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	
	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		resp, err := http.Get(testURL + "/clipboard")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		if i == 29 {
			cancel()
			return nil, fmt.Errorf("server did not start within timeout")
		}
	}
	
	return cancel, nil
}

func runClient(t *testing.T, args ...string) (string, error) {
	t.Helper()
	
	cmd := exec.Command("go", append([]string{"run", "cmd/client/main.go"}, args...)...)
	cmd.Env = append(os.Environ(), "REST_CLIPBOARD_URL="+testURL)
	
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func TestClientServerIntegration(t *testing.T) {
	cancel, err := startTestServer(t)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer cancel()
	
	// Test 1: Set and get basic text
	t.Run("SetAndGet", func(t *testing.T) {
		testText := "Hello, integration test!"
		
		// Set clipboard content
		_, err := runClient(t, "set", testText)
		if err != nil {
			t.Fatalf("Failed to set clipboard: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != testText {
			t.Errorf("Expected %q, got %q", testText, output)
		}
	})
	
	// Test 2: Set with custom device name
	t.Run("SetWithDevice", func(t *testing.T) {
		testText := "Device test"
		deviceName := "test-device"
		
		// Set clipboard content with device name
		_, err := runClient(t, "set", testText, deviceName)
		if err != nil {
			t.Fatalf("Failed to set clipboard with device: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != testText {
			t.Errorf("Expected %q, got %q", testText, output)
		}
	})
	
	// Test 3: Set from stdin (auto-detected)
	t.Run("SetFromStdinAuto", func(t *testing.T) {
		testText := "Stdin test content"
		
		// Set clipboard content from stdin (auto-detected)
		cmd := exec.Command("go", "run", "cmd/client/main.go", "set")
		cmd.Env = append(os.Environ(), "REST_CLIPBOARD_URL="+testURL)
		cmd.Stdin = strings.NewReader(testText)
		
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to set clipboard from stdin: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != testText {
			t.Errorf("Expected %q, got %q", testText, output)
		}
	})
	
	// Test 4: Set from stdin (explicit with -)
	t.Run("SetFromStdinExplicit", func(t *testing.T) {
		testText := "Explicit stdin test content"
		
		// Set clipboard content from stdin (explicit)
		cmd := exec.Command("go", "run", "cmd/client/main.go", "set", "-")
		cmd.Env = append(os.Environ(), "REST_CLIPBOARD_URL="+testURL)
		cmd.Stdin = strings.NewReader(testText)
		
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to set clipboard from stdin: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != testText {
			t.Errorf("Expected %q, got %q", testText, output)
		}
	})
	
	// Test 5: Empty clipboard
	t.Run("EmptyClipboard", func(t *testing.T) {
		// Set empty content
		_, err := runClient(t, "set", "")
		if err != nil {
			t.Fatalf("Failed to set empty clipboard: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != "" {
			t.Errorf("Expected empty string, got %q", output)
		}
	})
	
	// Test 6: Multiple operations
	t.Run("MultipleOperations", func(t *testing.T) {
		operations := []string{
			"First operation",
			"Second operation", 
			"Third operation with special chars: !@#$%^&*()",
		}
		
		for i, text := range operations {
			// Set clipboard content
			_, err := runClient(t, "set", text)
			if err != nil {
				t.Fatalf("Failed to set clipboard on operation %d: %v", i, err)
			}
			
			// Get clipboard content
			output, err := runClient(t, "get")
			if err != nil {
				t.Fatalf("Failed to get clipboard on operation %d: %v", i, err)
			}
			
			if output != text {
				t.Errorf("Operation %d: Expected %q, got %q", i, text, output)
			}
		}
	})
}

func TestClientServerErrorHandling(t *testing.T) {
	cancel, err := startTestServer(t)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer cancel()
	
	// Test invalid command
	t.Run("InvalidCommand", func(t *testing.T) {
		_, err := runClient(t, "invalid")
		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})
	
	// Test set without arguments
	t.Run("SetWithoutArgs", func(t *testing.T) {
		_, err := runClient(t, "set")
		if err == nil {
			t.Error("Expected error for set without arguments")
		}
	})
}

func TestClientEnvironmentVariables(t *testing.T) {
	cancel, err := startTestServer(t)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer cancel()
	
	// Test custom device name via environment variable
	t.Run("CustomDeviceEnv", func(t *testing.T) {
		testText := "Environment device test"
		
		// Set clipboard with custom device via env var
		cmd := exec.Command("go", "run", "cmd/client/main.go", "set", testText)
		cmd.Env = append(os.Environ(), 
			"REST_CLIPBOARD_URL="+testURL,
			"REST_CLIPBOARD_DEVICE=env-device")
		
		_, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to set clipboard with env device: %v", err)
		}
		
		// Get clipboard content
		output, err := runClient(t, "get")
		if err != nil {
			t.Fatalf("Failed to get clipboard: %v", err)
		}
		
		if output != testText {
			t.Errorf("Expected %q, got %q", testText, output)
		}
	})
}

func TestClientServerConnectionError(t *testing.T) {
	// Test connection to non-existent server
	t.Run("ConnectionError", func(t *testing.T) {
		cmd := exec.Command("go", "run", "cmd/client/main.go", "get")
		cmd.Env = append(os.Environ(), "REST_CLIPBOARD_URL=http://localhost:19999")
		
		_, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error when connecting to non-existent server")
		}
	})
}