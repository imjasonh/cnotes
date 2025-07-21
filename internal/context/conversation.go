package context

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

// ConversationContext represents relevant conversation context for a commit
type ConversationContext struct {
	UserPrompts      []string          `json:"user_prompts"`
	ClaudeResponses  []string          `json:"claude_responses"`
	ToolInteractions []ToolInteraction `json:"tool_interactions"`
}

// ToolInteraction represents a tool use and its result
type ToolInteraction struct {
	Tool     string `json:"tool"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Duration string `json:"duration,omitempty"`
}

// ContextExtractor extracts relevant conversation context from transcripts
type ContextExtractor struct {
	maxExcerptLength  int
	sensitivePatterns []*regexp.Regexp
}

// NewContextExtractor creates a new context extractor with default settings
func NewContextExtractor() *ContextExtractor {
	// Patterns to filter out sensitive information
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|token|key|secret)[:\s]*[^\s\n]+`),
		regexp.MustCompile(`(?i)(api[_-]?key)[:\s]*[^\s\n]+`),
		regexp.MustCompile(`-----BEGIN [A-Z ]+-----`),  // Private keys
		regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`), // Base64 encoded secrets
	}

	return &ContextExtractor{
		maxExcerptLength:  5000, // 5KB limit for context
		sensitivePatterns: sensitivePatterns,
	}
}

// ExtractRecentContext extracts recent conversation context from a transcript file
func (ce *ContextExtractor) ExtractRecentContext(transcriptPath string, sessionID string) (*ConversationContext, error) {
	return ce.ExtractContextSince(transcriptPath, sessionID, time.Time{})
}

// ExtractContextSince extracts conversation context since a given timestamp
func (ce *ContextExtractor) ExtractContextSince(transcriptPath string, sessionID string, since time.Time) (*ConversationContext, error) {
	if transcriptPath == "" {
		return &ConversationContext{}, nil
	}

	file, err := os.Open(transcriptPath)
	if err != nil {
		// Transcript might not exist, return empty context
		return &ConversationContext{}, nil
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	// Parse the transcript content
	context := ce.parseTranscriptContent(string(content), sessionID, since)

	// Apply privacy filters
	context = ce.filterSensitiveContent(context)

	return context, nil
}

// parseTranscriptContent parses transcript content and extracts conversation elements
func (ce *ContextExtractor) parseTranscriptContent(content, sessionID string, since time.Time) *ConversationContext {
	context := &ConversationContext{
		UserPrompts:      []string{},
		ClaudeResponses:  []string{},
		ToolInteractions: []ToolInteraction{},
	}

	lines := strings.Split(content, "\n")
	
	// Parse JSONL format
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse each line as JSON
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		// Only process entries for the current session
		entrySessionID, _ := entry["sessionId"].(string)
		if entrySessionID != "" && entrySessionID != sessionID {
			continue
		}

		// Filter by timestamp if provided
		if !since.IsZero() {
			if timestampStr, ok := entry["timestamp"].(string); ok {
				entryTime, err := time.Parse(time.RFC3339, timestampStr)
				if err == nil && entryTime.Before(since) {
					continue // Skip entries before the cutoff
				}
			}
		}

		// Extract based on type
		entryType, _ := entry["type"].(string)
		
		switch entryType {
		case "user":
			// Extract user prompts
			if msg, ok := entry["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].([]interface{}); ok {
					for _, c := range content {
						if textContent, ok := c.(map[string]interface{}); ok {
							if text, ok := textContent["text"].(string); ok && text != "" {
								// Skip system messages about interruptions
								if !strings.Contains(text, "[Request interrupted by user") {
									context.UserPrompts = append(context.UserPrompts, text)
								}
							}
						}
					}
				}
			}
			
		case "assistant":
			// Extract tool uses from assistant messages
			if msg, ok := entry["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].([]interface{}); ok {
					for _, c := range content {
						if toolUse, ok := c.(map[string]interface{}); ok {
							if toolUse["type"] == "tool_use" {
								toolName, _ := toolUse["name"].(string)
								if input, ok := toolUse["input"].(map[string]interface{}); ok {
									interaction := ToolInteraction{
										Tool: toolName,
									}
									
									// Extract key information based on tool type
									switch toolName {
									case "Bash":
										if cmd, ok := input["command"].(string); ok {
											interaction.Input = cmd
										}
									case "Write", "Edit", "MultiEdit":
										if path, ok := input["file_path"].(string); ok {
											interaction.Input = path
										}
									case "Read":
										if path, ok := input["file_path"].(string); ok {
											interaction.Input = path
										}
									case "WebFetch":
										if url, ok := input["url"].(string); ok {
											interaction.Input = url
										}
									default:
										// For other tools, try to get a meaningful representation
										if bytes, err := json.Marshal(input); err == nil {
											interaction.Input = string(bytes)
										}
									}
									
									if interaction.Input != "" {
										context.ToolInteractions = append(context.ToolInteractions, interaction)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return context
}


// filterSensitiveContent removes sensitive information from context
func (ce *ContextExtractor) filterSensitiveContent(context *ConversationContext) *ConversationContext {
	// Filter user prompts
	for i, prompt := range context.UserPrompts {
		context.UserPrompts[i] = ce.sanitizeText(prompt)
	}

	// Filter Claude responses
	for i, response := range context.ClaudeResponses {
		context.ClaudeResponses[i] = ce.sanitizeText(response)
	}

	// Filter tool interactions
	for i, interaction := range context.ToolInteractions {
		context.ToolInteractions[i].Input = ce.sanitizeText(interaction.Input)
		context.ToolInteractions[i].Output = ce.sanitizeText(interaction.Output)
	}

	return context
}

// sanitizeText removes sensitive patterns from text
func (ce *ContextExtractor) sanitizeText(text string) string {
	for _, pattern := range ce.sensitivePatterns {
		text = pattern.ReplaceAllString(text, "[REDACTED]")
	}
	return text
}

// CreateExcerpt creates a concise excerpt from conversation context
func (ce *ContextExtractor) CreateExcerpt(context *ConversationContext) string {
	var parts []string

	// Include all user prompts (they're already filtered by time)
	if len(context.UserPrompts) > 0 {
		parts = append(parts, "User prompts since last commit:")
		for _, prompt := range context.UserPrompts {
			if len(prompt) > 300 {
				prompt = prompt[:297] + "..."
			}
			parts = append(parts, fmt.Sprintf("- %s", prompt))
		}
	}

	// Include all tool interactions (they're already filtered by time)
	if len(context.ToolInteractions) > 0 {
		parts = append(parts, "\nTool interactions since last commit:")
		for _, interaction := range context.ToolInteractions {
			input := interaction.Input
			if len(input) > 100 {
				input = input[:97] + "..."
			}
			parts = append(parts, fmt.Sprintf("- %s: %s", interaction.Tool, input))
		}
	}

	excerpt := strings.Join(parts, "\n")

	// Truncate if too long
	if len(excerpt) > ce.maxExcerptLength {
		excerpt = excerpt[:ce.maxExcerptLength-3] + "..."
	}

	return excerpt
}
