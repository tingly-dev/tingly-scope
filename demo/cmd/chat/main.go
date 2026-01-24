package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	baseURL   = "http://localhost:12580/tingly/claude_code"
	modelName = "tingly/cc"
	apiToken  = "tingly-box-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJ0ZXN0LWNsaWVudCIsImV4cCI6MTc2NjQwMzQwNSwiaWF0IjoxNzY2MzE3MDA1fQ.AHtmsHxGGJ0jtzvrTZMHC3kfl3Os94HOhMA-zXFtHXQ"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestOptions struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		printHelp()
		return
	}

	// Check for single prompt mode
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		prompt := strings.Join(os.Args[1:], " ")
		response, err := sendRequest([]Message{
			{Role: "user", Content: prompt},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(response)
		return
	}

	// Interactive mode
	runInteractiveChat()
}

func printHelp() {
	fmt.Println("🤖 Tingly Chat - CLI Chat Assistant")
	fmt.Println("\nUsage:")
	fmt.Println("  tingly-chat <prompt>    # Single prompt mode")
	fmt.Println("  tingly-chat              # Interactive mode")
	fmt.Println("  tingly-chat --help       # Show this help message")
	fmt.Println("\nInteractive Commands:")
	fmt.Println("  /quit, /exit, /q  - Quit the chat")
	fmt.Println("  /clear, /c       - Clear conversation history")
	fmt.Println("  /help, /h        - Show this help message")
}

func runInteractiveChat() {
	fmt.Println("\n🤖 Tingly Chat - Interactive Mode")
	fmt.Println("Type your message and press Enter. Type /quit to exit.\n")

	messages := []Message{}
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Show prompt
		fmt.Print("\033[32m➜\033[0m ")

		// Read input
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			handled := handleCommand(input, &messages)
			if !handled {
				fmt.Println("❓ Unknown command. Type /help for available commands.")
			}
			continue
		}

		// Add user message
		messages = append(messages, Message{Role: "user", Content: input})

		// Show thinking indicator
		fmt.Print("\033[90mThinking...\033[0m\r")

		// Send request
		response, err := sendRequest(messages)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\r\033[31mError: %v\033[0m\n", err)
			continue
		}

		// Clear thinking indicator and print response
		fmt.Print("\r\033[K")
		fmt.Printf("\033[36m%s\033[0m\n", response) // Cyan color for assistant

		// Add assistant message to history
		messages = append(messages, Message{Role: "assistant", Content: response})
	}
}

func handleCommand(input string, messages *[]Message) bool {
	switch input {
	case "/quit", "/exit", "/q":
		fmt.Println("👋 Goodbye!")
		os.Exit(0)
	case "/clear", "/c":
		*messages = []Message{}
		fmt.Println("✨ Conversation history cleared.")
		return true
	case "/help", "/h":
		fmt.Println("\nCommands:")
		fmt.Println("  /quit, /exit, /q  - Quit the chat")
		fmt.Println("  /clear, /c       - Clear conversation history")
		fmt.Println("  /help, /h        - Show this help message")
		return true
	}
	return false
}

func sendRequest(messages []Message) (string, error) {
	reqBody := RequestOptions{
		Model:     modelName,
		MaxTokens: 4096,
		Messages:  messages,
		Stream:    false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST",
		baseURL+"/v1/messages",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiToken)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w (body: %s)", err, string(body))
	}

	var response strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			response.WriteString(block.Text)
		}
	}

	return response.String(), nil
}
