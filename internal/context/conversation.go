package context

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	
	"github.com/imjasonh/cnotes/internal/config"
)

// ConversationContext represents relevant conversation context for a commit
type ConversationContext struct {
	UserPrompts      []string          `json:"user_prompts"`
	ClaudeResponses  []string          `json:"claude_responses"`
	ToolInteractions []ToolInteraction `json:"tool_interactions"`
	Events           []ConversationEvent `json:"events"` // New: chronological events
	LastEventTime    time.Time         `json:"last_event_time"` // Track the latest event timestamp
}

// ConversationEvent represents any event in the conversation
type ConversationEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "user", "assistant", "tool", "system"
	Content   string    `json:"content"`
	ToolName  string    `json:"tool_name,omitempty"`
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
	config            *config.NotesConfig
}

// NewContextExtractor creates a new context extractor with default settings
func NewContextExtractor(cfg *config.NotesConfig) *ContextExtractor {
	// Patterns to filter out sensitive information
	sensitivePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|token|key|secret)[:\s]*[^\s\n]+`),
		regexp.MustCompile(`(?i)(api[_-]?key)[:\s]*[^\s\n]+`),
		regexp.MustCompile(`-----BEGIN [A-Z ]+-----`),  // Private keys
		regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`), // Base64 encoded secrets
	}

	maxLength := 5000
	if cfg != nil && cfg.MaxExcerptLength > 0 {
		maxLength = cfg.MaxExcerptLength
	}

	return &ContextExtractor{
		maxExcerptLength:  maxLength,
		sensitivePatterns: sensitivePatterns,
		config:            cfg,
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

	// Get the directory containing transcripts
	transcriptDir := filepath.Dir(transcriptPath)
	
	// Initialize combined context
	combinedContext := &ConversationContext{
		UserPrompts:      []string{},
		ClaudeResponses:  []string{},
		ToolInteractions: []ToolInteraction{},
		Events:           []ConversationEvent{},
	}

	// Read all transcript files in the directory
	files, err := os.ReadDir(transcriptDir)
	if err != nil {
		// If we can't read the directory, fall back to just the current transcript
		return ce.extractFromSingleTranscript(transcriptPath, sessionID, since)
	}

	// Process each transcript file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}
		
		filePath := filepath.Join(transcriptDir, file.Name())
		context, err := ce.extractFromSingleTranscript(filePath, "", since) // Empty sessionID to get all sessions
		if err != nil {
			continue // Skip files that can't be read
		}
		
		// Merge contexts
		combinedContext.UserPrompts = append(combinedContext.UserPrompts, context.UserPrompts...)
		combinedContext.ClaudeResponses = append(combinedContext.ClaudeResponses, context.ClaudeResponses...)
		combinedContext.ToolInteractions = append(combinedContext.ToolInteractions, context.ToolInteractions...)
		combinedContext.Events = append(combinedContext.Events, context.Events...)
	}

	// Apply privacy filters
	combinedContext = ce.filterSensitiveContent(combinedContext)

	return combinedContext, nil
}

// extractFromSingleTranscript extracts context from a single transcript file
func (ce *ContextExtractor) extractFromSingleTranscript(transcriptPath string, sessionID string, since time.Time) (*ConversationContext, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return &ConversationContext{}, nil
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript: %w", err)
	}

	// Parse the transcript content
	context := ce.parseTranscriptContent(string(content), sessionID, since)

	return context, nil
}

// parseTranscriptContent parses transcript content and extracts conversation elements
func (ce *ContextExtractor) parseTranscriptContent(content, sessionID string, since time.Time) *ConversationContext {
	context := &ConversationContext{
		UserPrompts:      []string{},
		ClaudeResponses:  []string{},
		ToolInteractions: []ToolInteraction{},
		Events:           []ConversationEvent{},
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

		// Only process entries for the current session (unless sessionID is empty)
		if sessionID != "" {
			entrySessionID, _ := entry["sessionId"].(string)
			if entrySessionID != "" && entrySessionID != sessionID {
				continue
			}
		}

		// Extract timestamp
		var entryTime time.Time
		if timestampStr, ok := entry["timestamp"].(string); ok {
			entryTime, _ = time.Parse(time.RFC3339, timestampStr)
		}
		
		// Filter by timestamp if provided
		if !since.IsZero() && !entryTime.IsZero() && entryTime.Before(since) {
			continue // Skip entries before the cutoff
		}

		// Extract based on type
		entryType, _ := entry["type"].(string)
		
		switch entryType {
		case "user":
			// Extract user prompts
			if msg, ok := entry["message"].(map[string]interface{}); ok {
				// Handle both string content and array content formats
				if content, ok := msg["content"].(string); ok && content != "" {
					// Direct string content
					if !strings.Contains(content, "[Request interrupted by user") {
						context.UserPrompts = append(context.UserPrompts, content)
						// Add to events
						context.Events = append(context.Events, ConversationEvent{
							Timestamp: entryTime,
							Type:      "user",
							Content:   content,
						})
					}
				} else if contentArray, ok := msg["content"].([]interface{}); ok {
					// Array of content objects
					for _, c := range contentArray {
						if textContent, ok := c.(map[string]interface{}); ok {
							if text, ok := textContent["text"].(string); ok && text != "" {
								// Skip system messages about interruptions
								if !strings.Contains(text, "[Request interrupted by user") {
									context.UserPrompts = append(context.UserPrompts, text)
									// Add to events
									context.Events = append(context.Events, ConversationEvent{
										Timestamp: entryTime,
										Type:      "user",
										Content:   text,
									})
								}
							}
						}
					}
				}
			}
			
		case "assistant":
			// Extract tool uses and text responses from assistant messages
			if msg, ok := entry["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].([]interface{}); ok {
					for _, c := range content {
						if contentItem, ok := c.(map[string]interface{}); ok {
							contentType, _ := contentItem["type"].(string)
							
							switch contentType {
							case "text":
								// Assistant text response
								if text, ok := contentItem["text"].(string); ok && text != "" {
									context.ClaudeResponses = append(context.ClaudeResponses, text)
									// Add to events
									context.Events = append(context.Events, ConversationEvent{
										Timestamp: entryTime,
										Type:      "assistant",
										Content:   text,
									})
								}
							
							case "tool_use":
								// Tool use
								toolName, _ := contentItem["name"].(string)
								if input, ok := contentItem["input"].(map[string]interface{}); ok {
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
										// Add to events
										context.Events = append(context.Events, ConversationEvent{
											Timestamp: entryTime,
											Type:      "tool",
											Content:   interaction.Input,
											ToolName:  toolName,
										})
									}
								}
							}
						}
					}
				}
			}
		
		case "tool_result":
			// Extract tool results
			if result, ok := entry["result"].(map[string]interface{}); ok {
				var resultContent string
				toolName, _ := entry["tool_name"].(string)
				
				if stdout, ok := result["stdout"].(string); ok && stdout != "" {
					resultContent = stdout
				} else if output, ok := result["output"].(string); ok && output != "" {
					resultContent = output
				}
				
				if resultContent != "" {
					// Add to events
					context.Events = append(context.Events, ConversationEvent{
						Timestamp: entryTime,
						Type:      "tool_result",
						Content:   resultContent,
						ToolName:  toolName,
					})
				}
			}
		}
	}

	// Track the last event time from all events
	for _, event := range context.Events {
		if event.Timestamp.After(context.LastEventTime) {
			context.LastEventTime = event.Timestamp
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
	// Sort events by timestamp
	sort.Slice(context.Events, func(i, j int) bool {
		return context.Events[i].Timestamp.Before(context.Events[j].Timestamp)
	})

	var parts []string
	for _, event := range context.Events {
		var line string
		
		switch event.Type {
		case "user":
			// Format user prompts
			content := event.Content
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			emoji := "👤"
			if ce.config != nil && ce.config.UserEmoji != "" {
				emoji = ce.config.UserEmoji
			}
			line = fmt.Sprintf("%s User: %s", emoji, content)
			
		case "assistant":
			// Format assistant responses
			content := event.Content
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			emoji := "🤖"
			if ce.config != nil && ce.config.AssistantEmoji != "" {
				emoji = ce.config.AssistantEmoji
			}
			line = fmt.Sprintf("%s Claude: %s", emoji, content)
			
		case "tool":
			// Format tool uses
			content := event.Content
			if len(content) > 150 {
				content = content[:147] + "..."
			}
			line = fmt.Sprintf("Tool (%s): %s", event.ToolName, content)
			
		case "tool_result":
			// Format tool results - show abbreviated output
			content := event.Content
			lines := strings.Split(content, "\n")
			if len(lines) > 3 {
				content = strings.Join(lines[:3], "\n") + "\n[...]"
			} else if len(content) > 150 {
				content = content[:147] + "..."
			}
			line = fmt.Sprintf("Result: %s", content)
		}
		
		if line != "" {
			parts = append(parts, line)
		}
	}

	excerpt := strings.Join(parts, "\n\n")

	// Truncate if too long
	if len(excerpt) > ce.maxExcerptLength {
		excerpt = excerpt[:ce.maxExcerptLength-3] + "..."
	}

	return excerpt
}
