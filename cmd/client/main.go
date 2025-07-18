package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type SetRequest struct {
	Text   string `json:"text"`
	Device string `json:"device"`
}

func getBaseURL() string {
	if url := os.Getenv("REST_CLIPBOARD_URL"); url != "" {
		return url
	}
	return "http://localhost:8080"
}

func getDeviceName() string {
	if device := os.Getenv("REST_CLIPBOARD_DEVICE"); device != "" {
		return device
	}
	return "cli"
}

func get() error {
	resp, err := http.Get(getBaseURL() + "/clipboard")
	if err != nil {
		return fmt.Errorf("failed to get clipboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Print(string(body))
	return nil
}

func set(text, device string) error {
	if device == "" {
		device = getDeviceName()
	}

	req := SetRequest{
		Text:   text,
		Device: device,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(getBaseURL()+"/clipboard", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to set clipboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	return string(data), nil
}

func isStdinAvailable() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  get                    - Get clipboard content\n")
	fmt.Fprintf(os.Stderr, "  set <text> [device]    - Set clipboard content\n")
	fmt.Fprintf(os.Stderr, "  set [device]           - Set clipboard content from stdin (auto-detected)\n")
	fmt.Fprintf(os.Stderr, "  set - [device]         - Set clipboard content from stdin (explicit)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
	fmt.Fprintf(os.Stderr, "  REST_CLIPBOARD_URL     - Server URL (default: http://localhost:8080)\n")
	fmt.Fprintf(os.Stderr, "  REST_CLIPBOARD_DEVICE  - Device name (default: cli)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s get\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s set \"hello world\"\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s set \"hello world\" laptop\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s set\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s set -\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s set laptop\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  REST_CLIPBOARD_URL=http://example.com:8080 %s get\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  REST_CLIPBOARD_DEVICE=laptop %s set \"hello world\"\n", os.Args[0])
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	command := os.Args[1]

	switch command {
	case "get":
		if err := get(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "set":
		var text string
		var device string
		var err error

		// Check if stdin has data available
		if len(os.Args) < 3 && isStdinAvailable() {
			// Read from stdin when no arguments provided but stdin has data
			text, err = readStdin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if len(os.Args) >= 3 && os.Args[2] == "-" {
			// Explicit stdin read with "-"
			text, err = readStdin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if len(os.Args) >= 4 {
				device = os.Args[3]
			}
		} else if len(os.Args) >= 3 {
			// Regular argument
			text = os.Args[2]
			if len(os.Args) >= 4 {
				device = os.Args[3]
			}
		} else {
			// No arguments and no stdin data
			usage()
		}

		if err := set(text, device); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		usage()
	}
}