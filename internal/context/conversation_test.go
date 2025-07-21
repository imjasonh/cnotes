package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/imjasonh/cnotes/internal/config"
)

func TestNewContextExtractor(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		ce := NewContextExtractor(nil)

		if ce.maxExcerptLength != 5000 {
			t.Errorf("expected default maxExcerptLength 5000, got %d", ce.maxExcerptLength)
		}

		if len(ce.sensitivePatterns) == 0 {
			t.Error("expected sensitive patterns to be initialized")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &config.NotesConfig{
			MaxExcerptLength: 1000,
		}

		ce := NewContextExtractor(cfg)

		if ce.maxExcerptLength != 1000 {
			t.Errorf("expected maxExcerptLength 1000, got %d", ce.maxExcerptLength)
		}

		if ce.config != cfg {
			t.Error("expected config to be set")
		}
	})
}

func TestParseTranscriptContent(t *testing.T) {
	ce := NewContextExtractor(nil)
	now := time.Now()

	// Create test transcript content
	entries := []map[string]interface{}{
		{
			"type":      "user",
			"sessionId": "test-session",
			"timestamp": now.Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": "Help me write a function",
			},
		},
		{
			"type":      "assistant",
			"sessionId": "test-session",
			"timestamp": now.Add(1 * time.Second).Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "I'll help you write a function.",
					},
					map[string]interface{}{
						"type": "tool_use",
						"name": "Write",
						"input": map[string]interface{}{
							"file_path": "test.go",
							"content":   "func Test() {}",
						},
					},
				},
			},
		},
		{
			"type":      "tool_result",
			"sessionId": "test-session",
			"timestamp": now.Add(2 * time.Second).Format(time.RFC3339),
			"tool_name": "Write",
			"result": map[string]interface{}{
				"output": "File created successfully",
			},
		},
	}

	// Convert to JSONL
	var lines []string
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		lines = append(lines, string(data))
	}
	content := strings.Join(lines, "\n")

	t.Run("parse all content", func(t *testing.T) {
		context := ce.parseTranscriptContent(content, "", time.Time{})

		if len(context.UserPrompts) != 1 {
			t.Errorf("expected 1 user prompt, got %d", len(context.UserPrompts))
		}

		if context.UserPrompts[0] != "Help me write a function" {
			t.Errorf("unexpected user prompt: %s", context.UserPrompts[0])
		}

		if len(context.ClaudeResponses) != 1 {
			t.Errorf("expected 1 Claude response, got %d", len(context.ClaudeResponses))
		}

		if len(context.ToolInteractions) != 1 {
			t.Errorf("expected 1 tool interaction, got %d", len(context.ToolInteractions))
		}

		if context.ToolInteractions[0].Tool != "Write" {
			t.Errorf("expected Write tool, got %s", context.ToolInteractions[0].Tool)
		}

		if len(context.Events) != 4 { // user, assistant text, tool use, tool result
			t.Errorf("expected 4 events, got %d", len(context.Events))
		}
	})

	t.Run("filter by session ID", func(t *testing.T) {
		context := ce.parseTranscriptContent(content, "test-session", time.Time{})

		if len(context.UserPrompts) != 1 {
			t.Errorf("expected 1 user prompt for session, got %d", len(context.UserPrompts))
		}

		// Try with different session ID
		context = ce.parseTranscriptContent(content, "other-session", time.Time{})

		if len(context.UserPrompts) != 0 {
			t.Errorf("expected 0 user prompts for other session, got %d", len(context.UserPrompts))
		}
	})

	t.Run("filter by timestamp", func(t *testing.T) {
		// Only get events after the first one
		context := ce.parseTranscriptContent(content, "", now.Add(30*time.Second))

		if len(context.UserPrompts) != 0 {
			t.Errorf("expected no user prompts after cutoff, got %d", len(context.UserPrompts))
		}

		if len(context.Events) != 0 {
			t.Errorf("expected no events after cutoff, got %d", len(context.Events))
		}
	})

	t.Run("handle array content format", func(t *testing.T) {
		// Test user message with array content
		arrayEntry := map[string]interface{}{
			"type":      "user",
			"timestamp": now.Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"text": "Array format message",
					},
				},
			},
		}

		data, _ := json.Marshal(arrayEntry)
		context := ce.parseTranscriptContent(string(data), "", time.Time{})

		if len(context.UserPrompts) != 1 {
			t.Errorf("expected 1 user prompt from array format, got %d", len(context.UserPrompts))
		}

		if context.UserPrompts[0] != "Array format message" {
			t.Errorf("unexpected array format message: %s", context.UserPrompts[0])
		}
	})

	t.Run("skip interrupted messages", func(t *testing.T) {
		interruptEntry := map[string]interface{}{
			"type": "user",
			"message": map[string]interface{}{
				"content": "[Request interrupted by user",
			},
		}

		data, _ := json.Marshal(interruptEntry)
		context := ce.parseTranscriptContent(string(data), "", time.Time{})

		if len(context.UserPrompts) != 0 {
			t.Error("expected interrupted message to be skipped")
		}
	})
}

func TestExtractToolInteractions(t *testing.T) {
	ce := NewContextExtractor(nil)

	toolUses := []map[string]interface{}{
		{
			"type": "tool_use",
			"name": "Bash",
			"input": map[string]interface{}{
				"command": "git commit -m \"test\"",
			},
		},
		{
			"type": "tool_use",
			"name": "Read",
			"input": map[string]interface{}{
				"file_path": "/path/to/file.txt",
			},
		},
		{
			"type": "tool_use",
			"name": "WebFetch",
			"input": map[string]interface{}{
				"url": "https://example.com",
			},
		},
		{
			"type": "tool_use",
			"name": "CustomTool",
			"input": map[string]interface{}{
				"param1": "value1",
				"param2": "value2",
			},
		},
	}

	// Create assistant message with tool uses
	entry := map[string]interface{}{
		"type": "assistant",
		"message": map[string]interface{}{
			"content": []interface{}{},
		},
	}

	// Add tool uses to content
	for _, toolUse := range toolUses {
		entry["message"].(map[string]interface{})["content"] = append(
			entry["message"].(map[string]interface{})["content"].([]interface{}),
			toolUse,
		)
	}

	data, _ := json.Marshal(entry)
	context := ce.parseTranscriptContent(string(data), "", time.Time{})

	if len(context.ToolInteractions) != 4 {
		t.Fatalf("expected 4 tool interactions, got %d", len(context.ToolInteractions))
	}

	// Check Bash tool
	if context.ToolInteractions[0].Tool != "Bash" {
		t.Errorf("expected Bash tool, got %s", context.ToolInteractions[0].Tool)
	}
	if context.ToolInteractions[0].Input != "git commit -m \"test\"" {
		t.Errorf("unexpected Bash input: %s", context.ToolInteractions[0].Input)
	}

	// Check Read tool
	if context.ToolInteractions[1].Input != "/path/to/file.txt" {
		t.Errorf("unexpected Read input: %s", context.ToolInteractions[1].Input)
	}

	// Check WebFetch tool
	if context.ToolInteractions[2].Input != "https://example.com" {
		t.Errorf("unexpected WebFetch input: %s", context.ToolInteractions[2].Input)
	}

	// Check custom tool (should have JSON representation)
	if !strings.Contains(context.ToolInteractions[3].Input, "param1") {
		t.Errorf("expected custom tool input to contain param1: %s", context.ToolInteractions[3].Input)
	}
}

func TestSanitizeText(t *testing.T) {
	ce := NewContextExtractor(nil)

	tests := []struct {
		name     string
		input    string
		contains []string // What the result should contain
		exact    string   // For exact match (optional)
	}{
		{
			name:     "password in text",
			input:    "my password: secret123",
			contains: []string{"[REDACTED]"},
		},
		{
			name:     "API key",
			input:    "API_KEY: abcd1234efgh5678",
			contains: []string{"[REDACTED]"},
		},
		{
			name:     "private key header",
			input:    "-----BEGIN RSA PRIVATE KEY-----\nkey content",
			contains: []string{"[REDACTED]"},
		},
		{
			name:     "base64 secret",
			input:    "token: " + strings.Repeat("A", 40) + "==",
			contains: []string{"[REDACTED]"},
		},
		{
			name:  "clean text",
			input: "This is clean text with no sensitive data",
			exact: "This is clean text with no sensitive data",
		},
		{
			name:     "multiple secrets",
			input:    "password: test123 and token: secret456",
			contains: []string{"[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ce.sanitizeText(tt.input)

			if tt.exact != "" {
				// Exact match
				if result != tt.exact {
					t.Errorf("expected exact %q, got %q", tt.exact, result)
				}
			} else {
				// Contains match
				for _, expected := range tt.contains {
					if !strings.Contains(result, expected) {
						t.Errorf("expected result to contain %q, got %q", expected, result)
					}
				}
			}
		})
	}
}

func TestCreateExcerpt(t *testing.T) {
	cfg := &config.NotesConfig{
		UserEmoji:      "👨",
		AssistantEmoji: "🤖",
	}
	ce := NewContextExtractor(cfg)

	now := time.Now()
	context := &ConversationContext{
		Events: []ConversationEvent{
			{
				Timestamp: now,
				Type:      "user",
				Content:   "Help me write a test",
			},
			{
				Timestamp: now.Add(1 * time.Second),
				Type:      "assistant",
				Content:   "I'll help you write a test. Let me create a test file for you.",
			},
			{
				Timestamp: now.Add(2 * time.Second),
				Type:      "tool",
				Content:   "test_example.go",
				ToolName:  "Write",
			},
			{
				Timestamp: now.Add(3 * time.Second),
				Type:      "tool_result",
				Content:   "File created successfully at test_example.go",
			},
		},
	}

	excerpt := ce.CreateExcerpt(context)

	// Check that all components are present
	if !strings.Contains(excerpt, "👨 User: Help me write a test") {
		t.Error("expected user prompt in excerpt")
	}

	if !strings.Contains(excerpt, "🤖 Claude:") {
		t.Error("expected Claude response in excerpt")
	}

	if !strings.Contains(excerpt, "Tool (Write): test_example.go") {
		t.Error("expected tool use in excerpt")
	}

	if !strings.Contains(excerpt, "Result: File created successfully") {
		t.Error("expected tool result in excerpt")
	}
}

func TestCreateExcerptTruncation(t *testing.T) {
	cfg := &config.NotesConfig{
		MaxExcerptLength: 100,
	}
	ce := NewContextExtractor(cfg)
	ce.maxExcerptLength = 100 // Ensure it's set

	// Create long content
	longContent := strings.Repeat("This is a very long message. ", 50)

	context := &ConversationContext{
		Events: []ConversationEvent{
			{
				Type:    "user",
				Content: longContent,
			},
		},
	}

	excerpt := ce.CreateExcerpt(context)

	if len(excerpt) > 100 {
		t.Errorf("expected excerpt length <= 100, got %d", len(excerpt))
	}

	if !strings.HasSuffix(excerpt, "...") {
		t.Error("expected truncated excerpt to end with ...")
	}
}

func TestExtractFromSingleTranscript(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-context-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test transcript file
	transcriptPath := filepath.Join(tempDir, "test.jsonl")

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"sessionId": "test-session",
			"message": map[string]interface{}{
				"content": "Test message",
			},
		},
	}

	// Write JSONL
	var lines []string
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		lines = append(lines, string(data))
	}

	if err := os.WriteFile(transcriptPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatalf("failed to write transcript: %v", err)
	}

	ce := NewContextExtractor(nil)

	t.Run("extract from existing file", func(t *testing.T) {
		context, err := ce.extractFromSingleTranscript(transcriptPath, "test-session", time.Time{})
		if err != nil {
			t.Fatalf("failed to extract: %v", err)
		}

		if len(context.UserPrompts) != 1 {
			t.Errorf("expected 1 user prompt, got %d", len(context.UserPrompts))
		}
	})

	t.Run("handle non-existent file", func(t *testing.T) {
		context, err := ce.extractFromSingleTranscript("/non/existent/file.jsonl", "", time.Time{})
		if err != nil {
			t.Fatalf("expected no error for non-existent file, got: %v", err)
		}

		if len(context.UserPrompts) != 0 {
			t.Error("expected empty context for non-existent file")
		}
	})
}

func TestExtractContextSince(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-multi-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	now := time.Now()

	// Create multiple transcript files
	for i := 0; i < 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("transcript%d.jsonl", i))

		entry := map[string]interface{}{
			"type":      "user",
			"timestamp": now.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": fmt.Sprintf("Message from file %d", i),
			},
		}

		data, _ := json.Marshal(entry)
		if err := os.WriteFile(filename, data, 0644); err != nil {
			t.Fatalf("failed to write transcript %d: %v", i, err)
		}
	}

	// Also create a non-JSONL file that should be skipped
	if err := os.WriteFile(filepath.Join(tempDir, "not-transcript.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatalf("failed to write non-transcript file: %v", err)
	}

	ce := NewContextExtractor(nil)

	t.Run("extract from all files", func(t *testing.T) {
		transcriptPath := filepath.Join(tempDir, "transcript0.jsonl")
		context, err := ce.ExtractContextSince(transcriptPath, "", time.Time{})
		if err != nil {
			t.Fatalf("failed to extract: %v", err)
		}

		// Should have messages from all 3 transcript files
		if len(context.UserPrompts) != 3 {
			t.Errorf("expected 3 user prompts from all files, got %d", len(context.UserPrompts))
		}

		// Check that all messages are present
		foundMessages := make(map[string]bool)
		for _, prompt := range context.UserPrompts {
			foundMessages[prompt] = true
		}

		for i := 0; i < 3; i++ {
			expected := fmt.Sprintf("Message from file %d", i)
			if !foundMessages[expected] {
				t.Errorf("missing message: %s", expected)
			}
		}
	})

	t.Run("filter by timestamp", func(t *testing.T) {
		transcriptPath := filepath.Join(tempDir, "transcript0.jsonl")
		// Only get messages after the first hour
		context, err := ce.ExtractContextSince(transcriptPath, "", now.Add(90*time.Minute))
		if err != nil {
			t.Fatalf("failed to extract: %v", err)
		}

		// Should only have messages from files 2 (after 90 minutes)
		if len(context.UserPrompts) != 1 {
			t.Errorf("expected 1 user prompt after cutoff, got %d", len(context.UserPrompts))
		}

		if context.UserPrompts[0] != "Message from file 2" {
			t.Errorf("unexpected message after cutoff: %s", context.UserPrompts[0])
		}
	})

	t.Run("handle empty transcript path", func(t *testing.T) {
		context, err := ce.ExtractContextSince("", "", time.Time{})
		if err != nil {
			t.Fatalf("unexpected error for empty path: %v", err)
		}

		if len(context.UserPrompts) != 0 {
			t.Error("expected empty context for empty path")
		}
	})
}

func TestFilterSensitiveContent(t *testing.T) {
	ce := NewContextExtractor(nil)

	context := &ConversationContext{
		UserPrompts: []string{
			"My password is secret123",
			"Clean prompt",
		},
		ClaudeResponses: []string{
			"Your API_KEY: abcd1234",
			"Clean response",
		},
		ToolInteractions: []ToolInteraction{
			{
				Tool:   "Bash",
				Input:  "export TOKEN=secret456",
				Output: "Token set",
			},
		},
	}

	filtered := ce.filterSensitiveContent(context)

	// Check user prompts
	if !strings.Contains(filtered.UserPrompts[0], "[REDACTED]") {
		t.Error("expected password to be redacted in user prompt")
	}

	if filtered.UserPrompts[1] != "Clean prompt" {
		t.Error("clean prompt should not be modified")
	}

	// Check Claude responses
	if !strings.Contains(filtered.ClaudeResponses[0], "[REDACTED]") {
		t.Error("expected API key to be redacted in Claude response")
	}

	// Check tool interactions
	if !strings.Contains(filtered.ToolInteractions[0].Input, "[REDACTED]") {
		t.Error("expected token to be redacted in tool input")
	}
}

func TestLastEventTimeTracking(t *testing.T) {
	ce := NewContextExtractor(nil)

	now := time.Now()
	laterTime := now.Add(10 * time.Minute)
	earlierTime := now.Add(-10 * time.Minute)

	entries := []map[string]interface{}{
		{
			"type":      "user",
			"timestamp": now.Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": "Middle message",
			},
		},
		{
			"type":      "user",
			"timestamp": laterTime.Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": "Latest message",
			},
		},
		{
			"type":      "user",
			"timestamp": earlierTime.Format(time.RFC3339),
			"message": map[string]interface{}{
				"content": "Earliest message",
			},
		},
	}

	// Convert to JSONL
	var lines []string
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		lines = append(lines, string(data))
	}
	content := strings.Join(lines, "\n")

	context := ce.parseTranscriptContent(content, "", time.Time{})

	// LastEventTime should be the latest timestamp
	// Use Unix() to compare seconds precision since parsing/formatting may lose nanoseconds
	if context.LastEventTime.Unix() != laterTime.Unix() {
		t.Errorf("expected LastEventTime to be %v, got %v", laterTime, context.LastEventTime)
	}
}
