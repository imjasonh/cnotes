package context

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
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
	context := ce.parseTranscriptContent(string(content), sessionID)

	// Apply privacy filters
	context = ce.filterSensitiveContent(context)

	return context, nil
}

// parseTranscriptContent parses transcript content and extracts conversation elements
func (ce *ContextExtractor) parseTranscriptContent(content, sessionID string) *ConversationContext {
	context := &ConversationContext{
		UserPrompts:      []string{},
		ClaudeResponses:  []string{},
		ToolInteractions: []ToolInteraction{},
	}

	lines := strings.Split(content, "\n")
	currentMessage := ""
	messageType := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Detect message boundaries and types
		if strings.HasPrefix(line, "Human:") || strings.HasPrefix(line, "User:") {
			// Save previous message if exists
			if currentMessage != "" && messageType == "claude" {
				context.ClaudeResponses = append(context.ClaudeResponses, strings.TrimSpace(currentMessage))
			}

			messageType = "human"
			currentMessage = strings.TrimPrefix(strings.TrimPrefix(line, "Human:"), "User:")
		} else if strings.HasPrefix(line, "Assistant:") || strings.HasPrefix(line, "Claude:") {
			// Save previous message if exists
			if currentMessage != "" && messageType == "human" {
				context.UserPrompts = append(context.UserPrompts, strings.TrimSpace(currentMessage))
			}

			messageType = "claude"
			currentMessage = strings.TrimPrefix(strings.TrimPrefix(line, "Assistant:"), "Claude:")
		} else if strings.Contains(line, "tool_use") || strings.Contains(line, "function_calls") {
			// Try to parse tool interactions
			toolInteraction := ce.parseToolInteraction(line)
			if toolInteraction != nil {
				context.ToolInteractions = append(context.ToolInteractions, *toolInteraction)
			}
		} else {
			// Continue building current message
			if currentMessage != "" {
				currentMessage += "\n"
			}
			currentMessage += line
		}
	}

	// Don't forget the last message
	if currentMessage != "" {
		if messageType == "human" {
			context.UserPrompts = append(context.UserPrompts, strings.TrimSpace(currentMessage))
		} else if messageType == "claude" {
			context.ClaudeResponses = append(context.ClaudeResponses, strings.TrimSpace(currentMessage))
		}
	}

	return context
}

// parseToolInteraction attempts to parse tool interaction from a line
func (ce *ContextExtractor) parseToolInteraction(line string) *ToolInteraction {
	// This is a simplified parser - in reality, the transcript format may vary
	if strings.Contains(line, "Bash") && strings.Contains(line, "command") {
		return &ToolInteraction{
			Tool:   "Bash",
			Input:  ce.extractBetween(line, "command", "description"),
			Output: "Command executed",
		}
	}

	if strings.Contains(line, "Edit") || strings.Contains(line, "Write") {
		return &ToolInteraction{
			Tool:   "File",
			Input:  ce.extractBetween(line, "file_path", "content"),
			Output: "File modified",
		}
	}

	return nil
}

// extractBetween extracts text between two markers
func (ce *ContextExtractor) extractBetween(text, start, end string) string {
	startIdx := strings.Index(text, start)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(text[startIdx:], end)
	if endIdx == -1 {
		return text[startIdx:]
	}

	return text[startIdx : startIdx+endIdx]
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

	// Include recent user prompts (last 2)
	if len(context.UserPrompts) > 0 {
		start := len(context.UserPrompts) - 2
		if start < 0 {
			start = 0
		}
		parts = append(parts, "Recent user prompts:")
		for i := start; i < len(context.UserPrompts); i++ {
			prompt := context.UserPrompts[i]
			if len(prompt) > 200 {
				prompt = prompt[:197] + "..."
			}
			parts = append(parts, fmt.Sprintf("- %s", prompt))
		}
	}

	// Include recent tool interactions
	if len(context.ToolInteractions) > 0 {
		parts = append(parts, "\nTool interactions:")
		for _, interaction := range context.ToolInteractions {
			parts = append(parts, fmt.Sprintf("- %s: %s", interaction.Tool, interaction.Input))
		}
	}

	excerpt := strings.Join(parts, "\n")

	// Truncate if too long
	if len(excerpt) > ce.maxExcerptLength {
		excerpt = excerpt[:ce.maxExcerptLength-3] + "..."
	}

	return excerpt
}
