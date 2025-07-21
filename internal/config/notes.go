package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// NotesConfig controls git notes behavior for conversation logging
type NotesConfig struct {
	Enabled           bool     `json:"enabled"`             // Whether to attach notes to commits
	MaxExcerptLength  int      `json:"max_excerpt_length"`  // Maximum length of conversation excerpt
	MaxPrompts        int      `json:"max_prompts"`         // Maximum number of user prompts to include
	IncludeToolOutput bool     `json:"include_tool_output"` // Whether to include tool output in notes
	NotesRef          string   `json:"notes_ref"`           // Git notes reference name
	ExcludePatterns   []string `json:"exclude_patterns"`    // Patterns to exclude from notes
	UserEmoji         string   `json:"user_emoji"`          // Emoji to use for user messages
	AssistantEmoji    string   `json:"assistant_emoji"`     // Emoji to use for assistant messages
}

// DefaultNotesConfig returns the default configuration
func DefaultNotesConfig() *NotesConfig {
	return &NotesConfig{
		Enabled:           true,
		MaxExcerptLength:  5000,
		MaxPrompts:        2,
		IncludeToolOutput: false, // Privacy: don't include potentially sensitive output
		NotesRef:          "claude-conversations",
		ExcludePatterns: []string{
			"password",
			"token",
			"key",
			"secret",
			"api_key",
			"auth",
		},
		UserEmoji:      "👤",
		AssistantEmoji: "🤖",
	}
}

// LoadNotesConfig loads notes configuration from file or returns default
func LoadNotesConfig(projectDir string) *NotesConfig {
	configPath := filepath.Join(projectDir, ".claude", "notes.json")

	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// File doesn't exist, use defaults
		return DefaultNotesConfig()
	}

	var config NotesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// Invalid config file, use defaults
		return DefaultNotesConfig()
	}

	// Ensure required fields have defaults if missing
	if config.NotesRef == "" {
		config.NotesRef = "claude-conversations"
	}
	if config.MaxExcerptLength <= 0 {
		config.MaxExcerptLength = 5000
	}
	if config.MaxPrompts <= 0 {
		config.MaxPrompts = 2
	}
	if config.UserEmoji == "" {
		config.UserEmoji = "👤"
	}
	if config.AssistantEmoji == "" {
		config.AssistantEmoji = "🤖"
	}

	return &config
}

// SaveNotesConfig saves notes configuration to file
func SaveNotesConfig(projectDir string, config *NotesConfig) error {
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(claudeDir, "notes.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
