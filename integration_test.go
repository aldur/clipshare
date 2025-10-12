package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	cmd.Env = append(os.Environ(), "CLIPSHARE_URL="+testURL)
	
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
		cmd.Env = append(os.Environ(), "CLIPSHARE_URL="+testURL)
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
		cmd.Env = append(os.Environ(), "CLIPSHARE_URL="+testURL)
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
			"CLIPSHARE_URL="+testURL,
			"CLIPSHARE_DEVICE=env-device")
		
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
		cmd.Env = append(os.Environ(), "CLIPSHARE_URL=http://localhost:19999")

		_, err := cmd.CombinedOutput()
		if err == nil {
			t.Error("Expected error when connecting to non-existent server")
		}
	})
}

func TestWebInterfaceTemplateRendering(t *testing.T) {
	cancel, err := startTestServer(t)
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer cancel()

	// Test 1: Empty clipboard renders correctly
	t.Run("EmptyClipboardTemplate", func(t *testing.T) {
		// Set empty clipboard
		_, err := runClient(t, "set", "")
		if err != nil {
			t.Fatalf("Failed to set empty clipboard: %v", err)
		}

		// Fetch the web interface
		resp, err := http.Get(testURL + "/")
		if err != nil {
			t.Fatalf("Failed to fetch web interface: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		html := string(body)

		// Should contain the textarea with empty content
		if !strings.Contains(html, `<textarea id="clipboardContent" readonly placeholder="Clipboard content will appear here..."></textarea>`) {
			t.Error("Expected empty textarea in HTML")
		}
	})

	// Test 2: Content in clipboard is rendered
	t.Run("ClipboardContentRendered", func(t *testing.T) {
		testText := "Hello from template test!"

		// Set clipboard content
		_, err := runClient(t, "set", testText)
		if err != nil {
			t.Fatalf("Failed to set clipboard: %v", err)
		}

		// Fetch the web interface
		resp, err := http.Get(testURL + "/")
		if err != nil {
			t.Fatalf("Failed to fetch web interface: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		html := string(body)

		// Should contain the test text inside the textarea
		expectedTextarea := `<textarea id="clipboardContent" readonly placeholder="Clipboard content will appear here...">` + testText + `</textarea>`
		if !strings.Contains(html, expectedTextarea) {
			t.Errorf("Expected textarea with content %q in HTML, got:\n%s", testText, html)
		}
	})

	// Test 3: HTML special characters are escaped
	t.Run("HTMLEscaping", func(t *testing.T) {
		testText := "<script>alert('xss')</script>"

		// Set clipboard content with HTML
		_, err := runClient(t, "set", testText)
		if err != nil {
			t.Fatalf("Failed to set clipboard: %v", err)
		}

		// Fetch the web interface
		resp, err := http.Get(testURL + "/")
		if err != nil {
			t.Fatalf("Failed to fetch web interface: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		html := string(body)

		// Should contain escaped HTML
		if !strings.Contains(html, "&lt;script&gt;") {
			t.Error("Expected HTML to be escaped in template")
		}
		// Should NOT contain the raw script tag
		if bytes.Count(body, []byte("<script>")) > 1 {
			t.Error("Expected script tag to be escaped, but found raw script tag")
		}
	})

	// Test 4: Multiline content is preserved
	t.Run("MultilineContent", func(t *testing.T) {
		testText := "Line 1\nLine 2\nLine 3"

		// Set clipboard content with newlines
		_, err := runClient(t, "set", testText)
		if err != nil {
			t.Fatalf("Failed to set clipboard: %v", err)
		}

		// Fetch the web interface
		resp, err := http.Get(testURL + "/")
		if err != nil {
			t.Fatalf("Failed to fetch web interface: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		html := string(body)

		// Should contain the multiline text
		if !strings.Contains(html, "Line 1\nLine 2\nLine 3") {
			t.Error("Expected multiline content to be preserved in template")
		}
	})
}
