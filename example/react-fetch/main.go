package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	apiURL   = "http://localhost:12580/tingly/claude_code"
	apiToken = "tingly-box-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaWQiOiJ0ZXN0LWNsaWVudCIsImV4cCI6MTc2NjQwMzQwNSwiaWF0IjoxNzY2MzE3MDA1fQ.AHtmsHxGGJ0jtzvrTZMHC3kfl3Os94HOhMA-zXFtHXQ"
	model    = "tingly/cc"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// API Request/Response structures
type ChatRequest struct {
	Model      string    `json:"model"`
	MaxTokens  int       `json:"max_tokens"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice string    `json:"tool_choice,omitempty"`
}

type Tool struct {
	Type     string         `json:"type"`
	Function map[string]any `json:"function"`
}

type ContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

type ChatResponse struct {
	Content []ContentBlock `json:"content"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		printHelp()
		return
	}

	fmt.Println("🤖 ReAct Agent Demo - Web Fetch Tool")
	fmt.Println("This agent can fetch web pages and answer questions about them.\n")

	// Example queries
	examples := []string{
		"what's the title of https://example.com?",
		"fetch https://www.python.org and tell me the latest Python version",
		"what's the first heading on https://www.ietf.org?",
		"/quit",
	}

	fmt.Println("Example queries you can try:")
	for _, ex := range examples {
		fmt.Printf("  • %s\n", ex)
	}
	fmt.Println()

	// Run interactive ReAct loop
	runReActLoop()
}

func printHelp() {
	fmt.Println("🤖 ReAct Agent Demo - Web Fetch Tool")
	fmt.Println("\nAn agent that uses the ReAct (Reasoning + Acting) pattern")
	fmt.Println("to answer questions by fetching web pages.\n")
	fmt.Println("Usage:")
	fmt.Println("  react-fetch           # Start interactive mode")
	fmt.Println("  react-fetch --help    # Show this help")
}

func runReActLoop() {
	agent := NewReActAgent()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[32m➜\033[0m ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "/quit" || input == "/exit" || input == "/q" {
			fmt.Println("👋 Goodbye!")
			return
		}

		if input == "/help" || input == "/h" {
			fmt.Println("\nCommands:")
			fmt.Println("  /quit, /exit, /q - Exit")
			fmt.Println("  Ask a question about a webpage!")
			continue
		}

		// Run ReAct loop
		response, err := agent.Run(context.Background(), input)
		if err != nil {
			fmt.Printf("\033[31mError: %v\033[0m\n", err)
			continue
		}

		fmt.Printf("\033[36m%s\033[0m\n", response)
	}
}

// ReActAgent implements the ReAct pattern
type ReActAgent struct {
	messages []Message
	maxTurns int
	tools    map[string]ToolFunc
}

type ToolFunc func(ctx context.Context, params map[string]any) (string, error)

// NewReActAgent creates a new ReAct agent
func NewReActAgent() *ReActAgent {
	agent := &ReActAgent{
		messages: []Message{},
		maxTurns: 5,
		tools:    make(map[string]ToolFunc),
	}

	// Register tools
	agent.RegisterTool("web_fetch", ToolFetch)

	// Set system prompt
	agent.messages = append(agent.messages, Message{
		Role: "user",
		Content: `You are a helpful assistant with access to a web_fetch tool.
When you need to get information from a URL, use the web_fetch tool.
After fetching, answer the user's question based on the fetched content.`,
	})

	return agent
}

// RegisterTool registers a tool function
func (a *ReActAgent) RegisterTool(name string, fn ToolFunc) {
	a.tools[name] = fn
}

// Run executes the ReAct loop
func (a *ReActAgent) Run(ctx context.Context, query string) (string, error) {
	// Add user message
	a.messages = append(a.messages, Message{
		Role:    "user",
		Content: query,
	})

	// ReAct loop
	for turn := 0; turn < a.maxTurns; turn++ {
		fmt.Printf("\033[90m[Turn %d] Thinking...\033[0m\r", turn+1)

		// Prepare tools for API
		tools := a.getToolDefinitions()

		// Call model
		response, err := a.callModel(ctx, tools)
		if err != nil {
			return "", err
		}

		// Check for tool use
		toolUses := a.extractToolUses(response)
		if len(toolUses) == 0 {
			// No tool use, return the answer
			fmt.Print("\r\033[K")
			return response, nil
		}

		// Execute tools
		fmt.Print("\r\033[K") // Clear thinking indicator

		for _, toolUse := range toolUses {
			fmt.Printf("\033[33m[Tool: %s]\033[0m ", toolUse.Name)

			// Execute tool
			result, err := a.executeTool(ctx, toolUse)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			fmt.Printf("→ %s\n", truncate(result, 100))

			// Add tool result to messages
			a.messages = append(a.messages, Message{
				Role:    "user",
				Content: fmt.Sprintf("Tool %s returned: %s", toolUse.Name, result),
			})
		}
	}

	return "", fmt.Errorf("max turns reached without final answer")
}

func (a *ReActAgent) getToolDefinitions() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: map[string]any{
				"name":        "web_fetch",
				"description": "Fetch and return the content of a webpage. Use this when you need to get information from a URL.",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "The URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		},
	}
}

func (a *ReActAgent) callModel(ctx context.Context, tools []Tool) (string, error) {
	req := ChatRequest{
		Model:      model,
		MaxTokens:  4096,
		Messages:   a.messages,
		Tools:      tools,
		ToolChoice: "auto",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		apiURL+"/v1/messages",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiToken)

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("parse error: %w (body: %s)", err, string(body))
	}

	// Extract text content
	var text strings.Builder
	for _, block := range chatResp.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return text.String(), nil
}

type ToolUse struct {
	ID    string
	Name  string
	Input map[string]any
}

func (a *ReActAgent) extractToolUses(response string) []ToolUse {
	// Parse tool use blocks from response
	// Pattern: <tool_name>(input)
	var toolUses []ToolUse

	// Also look for structured tool use in the response
	if strings.Contains(response, "web_fetch") {
		// Extract URL from the response
		urlRe := regexp.MustCompile(`(?:fetch|get|from)?\s*["']?(https?://[^"'\s]+)["']?`)
		matches := urlRe.FindStringSubmatch(response)
		if len(matches) > 1 {
			toolUses = append(toolUses, ToolUse{
				Name:  "web_fetch",
				Input: map[string]any{"url": matches[1]},
			})
			return toolUses
		}

		// Try to extract URL from JSON-like structures
		jsonRe := regexp.MustCompile(`"url"\s*:\s*"([^"]+)"`)
		jsonMatches := jsonRe.FindStringSubmatch(response)
		if len(jsonMatches) > 1 {
			toolUses = append(toolUses, ToolUse{
				Name:  "web_fetch",
				Input: map[string]any{"url": jsonMatches[1]},
			})
			return toolUses
		}
	}

	// Check if response indicates no tool needed
	if strings.Contains(strings.ToLower(response), "based on") ||
		strings.Contains(strings.ToLower(response), "the answer is") ||
		strings.Contains(strings.ToLower(response), "according to") {
		return []ToolUse{}
	}

	return toolUses
}

func (a *ReActAgent) executeTool(ctx context.Context, toolUse ToolUse) (string, error) {
	fn, ok := a.tools[toolUse.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", toolUse.Name)
	}

	return fn(ctx, toolUse.Input)
}

// ToolFetch fetches a webpage and returns its content
func ToolFetch(ctx context.Context, params map[string]any) (string, error) {
	urlStr, ok := params["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing url parameter")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL must start with http:// or https://")
	}

	// Fetch the page
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "ReActAgent/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	// Extract meaningful content
	content := string(body)
	content = cleanHTML(content)

	// Try to extract title
	title := extractTitle(content)

	// Try to extract main content (first paragraph, headings)
	mainContent := extractMainContent(content)

	return fmt.Sprintf("Title: %s\n\nContent: %s", title, mainContent), nil
}

func cleanHTML(content string) string {
	// Remove script and style tags
	scriptRe := regexp.MustCompile(`<script[^>]*>.*?</script>|<style[^>]*>.*?</style>`)
	content = scriptRe.ReplaceAllString(content, "")

	// Replace multiple whitespaces with single space
	spaceRe := regexp.MustCompile(`\s+`)
	content = spaceRe.ReplaceAllString(content, " ")

	// Decode HTML entities
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&quot", "\"")

	return strings.TrimSpace(content)
}

func extractTitle(content string) string {
	// Try title tag
	titleRe := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	matches := titleRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try h1 tag
	h1Re := regexp.MustCompile(`<h1[^>]*>(.*?)</h1>`)
	matches = h1Re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return cleanHTML(matches[1])
	}

	return "Unknown Title"
}

func extractMainContent(content string) string {
	// Extract first paragraph
	pRe := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	matches := pRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		text := cleanHTML(matches[1])
		if len(text) > 50 {
			return text
		}
	}

	// Extract first heading
	headingRe := regexp.MustCompile(`<h[1-6][^>]*>(.*?)</h[1-6]>`)
	matches = headingRe.FindStringSubmatch(content)
	if len(matches) > 1 {
		return cleanHTML(matches[1])
	}

	// Return first 500 chars
	return truncate(cleanHTML(content), 500)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
