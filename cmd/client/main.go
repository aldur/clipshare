package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

type SetRequest struct {
	Text   string `json:"text"`
	Device string `json:"device"`
}

var (
	url    string
	device string
)

func get(url string) error {
	resp, err := http.Get(url + "/clipboard")
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

func set(text, device, url string) error {

	req := SetRequest{
		Text:   text,
		Device: device,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(url+"/clipboard", "application/json", bytes.NewBuffer(jsonData))
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
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command> [args]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  get                    - Get clipboard content\n")
	fmt.Fprintf(os.Stderr, "  set <text>             - Set clipboard content\n")
	fmt.Fprintf(os.Stderr, "  set                    - Set clipboard content from stdin (auto-detected)\n")
	fmt.Fprintf(os.Stderr, "  set -                  - Set clipboard content from stdin (explicit)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
	fmt.Fprintf(os.Stderr, "  CLIPSHARE_URL     - Server URL (default: http://localhost:8080)\n")
	fmt.Fprintf(os.Stderr, "  CLIPSHARE_DEVICE  - Device name (default: cli)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s get\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s set \"hello world\"\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -device laptop set \"hello world\"\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s set\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s set -\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  echo \"hello world\" | %s -device laptop set\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -url http://example.com:8080 get\n", os.Args[0])
	os.Exit(1)
}

func main() {
	defaultURL := "http://localhost:8080"
	if urlEnv := os.Getenv("CLIPSHARE_URL"); urlEnv != "" {
		defaultURL = urlEnv
	}
	urlUsage := "Server URL"

	defaultDevice := "cli"
	if deviceEnv := os.Getenv("CLIPSHARE_DEVICE"); deviceEnv != "" {
		defaultDevice = deviceEnv
	}
	deviceUsage := "Device name"

	shorthand := " (shorthand)"

	flag.StringVar(&url, "url", defaultURL, urlUsage)
	flag.StringVar(&url, "u", defaultURL, urlUsage+shorthand)
	flag.StringVar(&device, "device", defaultDevice, deviceUsage)
	flag.StringVar(&device, "d", defaultDevice, deviceUsage+shorthand)

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
	}

	command := flag.Arg(0)

	switch command {
	case "get":
		if err := get(url); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "set":
		var text string
		var err error

		// Check if stdin has data available
		if flag.NArg() < 2 && isStdinAvailable() {
			// Read from stdin when no arguments provided but stdin has data
			text, err = readStdin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if flag.NArg() >= 2 && flag.Arg(1) == "-" {
			// Explicit stdin read with "-"
			text, err = readStdin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else if flag.NArg() >= 2 {
			// Regular argument
			text = flag.Arg(1)
		} else {
			// No arguments and no stdin data
			usage()
		}

		if err := set(text, device, url); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		usage()
	}
}
