package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultNotesConfig(t *testing.T) {
	config := DefaultNotesConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}

	if config.MaxExcerptLength != 5000 {
		t.Errorf("expected MaxExcerptLength 5000, got %d", config.MaxExcerptLength)
	}

	if config.MaxPrompts != 100 {
		t.Errorf("expected MaxPrompts 100, got %d", config.MaxPrompts)
	}

	if config.IncludeToolOutput {
		t.Error("expected IncludeToolOutput to be false by default")
	}

	if config.NotesRef != "claude-conversations" {
		t.Errorf("expected NotesRef claude-conversations, got %s", config.NotesRef)
	}

	expectedPatterns := []string{"password", "token", "key", "secret", "api_key", "auth"}
	if !reflect.DeepEqual(config.ExcludePatterns, expectedPatterns) {
		t.Errorf("unexpected exclude patterns: %v", config.ExcludePatterns)
	}

	if config.UserEmoji != "👤" {
		t.Errorf("expected UserEmoji 👤, got %s", config.UserEmoji)
	}

	if config.AssistantEmoji != "🤖" {
		t.Errorf("expected AssistantEmoji 🤖, got %s", config.AssistantEmoji)
	}
}

func TestLoadNotesConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cnotes-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("no config file", func(t *testing.T) {
		// When no config file exists, should return defaults
		config := LoadNotesConfig(tempDir)

		if !config.Enabled {
			t.Error("expected default Enabled to be true")
		}

		if config.MaxExcerptLength != 5000 {
			t.Errorf("expected default MaxExcerptLength 5000, got %d", config.MaxExcerptLength)
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Create custom config
		customConfig := &NotesConfig{
			Enabled:           false,
			MaxExcerptLength:  1000,
			MaxPrompts:        5,
			IncludeToolOutput: true,
			NotesRef:          "custom-notes",
			ExcludePatterns:   []string{"custom-pattern"},
			UserEmoji:         "🧑",
			AssistantEmoji:    "🤖",
		}

		// Write config file
		configPath := filepath.Join(claudeDir, "notes.json")
		data, _ := json.MarshalIndent(customConfig, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		// Load config
		config := LoadNotesConfig(tempDir)

		if config.Enabled != false {
			t.Error("expected Enabled to be false")
		}

		if config.MaxExcerptLength != 1000 {
			t.Errorf("expected MaxExcerptLength 1000, got %d", config.MaxExcerptLength)
		}

		if config.MaxPrompts != 5 {
			t.Errorf("expected MaxPrompts 5, got %d", config.MaxPrompts)
		}

		if !config.IncludeToolOutput {
			t.Error("expected IncludeToolOutput to be true")
		}

		if config.NotesRef != "custom-notes" {
			t.Errorf("expected NotesRef custom-notes, got %s", config.NotesRef)
		}

		if len(config.ExcludePatterns) != 1 || config.ExcludePatterns[0] != "custom-pattern" {
			t.Errorf("unexpected exclude patterns: %v", config.ExcludePatterns)
		}

		if config.UserEmoji != "🧑" {
			t.Errorf("expected UserEmoji 🧑, got %s", config.UserEmoji)
		}
	})

	t.Run("invalid JSON config", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Write invalid JSON
		configPath := filepath.Join(claudeDir, "notes.json")
		if err := os.WriteFile(configPath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		// Should return defaults when JSON is invalid
		config := LoadNotesConfig(tempDir)

		if !config.Enabled {
			t.Error("expected default Enabled to be true")
		}

		if config.MaxExcerptLength != 5000 {
			t.Errorf("expected default MaxExcerptLength 5000, got %d", config.MaxExcerptLength)
		}
	})

	t.Run("partial config with missing fields", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Create partial config (missing some fields)
		partialConfig := map[string]interface{}{
			"enabled": false,
			// Missing other fields
		}

		// Write config file
		configPath := filepath.Join(claudeDir, "notes.json")
		data, _ := json.MarshalIndent(partialConfig, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		// Load config
		config := LoadNotesConfig(tempDir)

		// Should have custom value for enabled
		if config.Enabled != false {
			t.Error("expected Enabled to be false")
		}

		// Should have defaults for missing fields
		if config.NotesRef != "claude-conversations" {
			t.Errorf("expected default NotesRef, got %s", config.NotesRef)
		}

		if config.MaxExcerptLength != 5000 {
			t.Errorf("expected default MaxExcerptLength, got %d", config.MaxExcerptLength)
		}

		if config.MaxPrompts != 2 { // Note: defaults to 2 when 0
			t.Errorf("expected default MaxPrompts 2, got %d", config.MaxPrompts)
		}

		if config.UserEmoji != "👤" {
			t.Errorf("expected default UserEmoji, got %s", config.UserEmoji)
		}

		if config.AssistantEmoji != "🤖" {
			t.Errorf("expected default AssistantEmoji, got %s", config.AssistantEmoji)
		}
	})

	t.Run("config with zero values", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Create config with zero/empty values
		zeroConfig := &NotesConfig{
			Enabled:          true,
			MaxExcerptLength: 0,  // Should be replaced with default
			MaxPrompts:       -1, // Should be replaced with default
			NotesRef:         "", // Should be replaced with default
			UserEmoji:        "", // Should be replaced with default
			AssistantEmoji:   "", // Should be replaced with default
		}

		// Write config file
		configPath := filepath.Join(claudeDir, "notes.json")
		data, _ := json.MarshalIndent(zeroConfig, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		// Load config
		config := LoadNotesConfig(tempDir)

		// Should have defaults for zero/empty values
		if config.NotesRef != "claude-conversations" {
			t.Errorf("expected default NotesRef, got %s", config.NotesRef)
		}

		if config.MaxExcerptLength != 5000 {
			t.Errorf("expected default MaxExcerptLength, got %d", config.MaxExcerptLength)
		}

		if config.MaxPrompts != 2 {
			t.Errorf("expected default MaxPrompts, got %d", config.MaxPrompts)
		}

		if config.UserEmoji != "👤" {
			t.Errorf("expected default UserEmoji, got %s", config.UserEmoji)
		}

		if config.AssistantEmoji != "🤖" {
			t.Errorf("expected default AssistantEmoji, got %s", config.AssistantEmoji)
		}
	})
}

func TestSaveNotesConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cnotes-save-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("save to new directory", func(t *testing.T) {
		config := &NotesConfig{
			Enabled:          false,
			MaxExcerptLength: 1500,
			MaxPrompts:       10,
			NotesRef:         "test-ref",
			ExcludePatterns:  []string{"test-pattern"},
			UserEmoji:        "👨",
			AssistantEmoji:   "🤖",
		}

		// Save config
		if err := SaveNotesConfig(tempDir, config); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		// Verify .claude directory was created
		claudeDir := filepath.Join(tempDir, ".claude")
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			t.Error(".claude directory was not created")
		}

		// Verify config file exists
		configPath := filepath.Join(claudeDir, "notes.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Error("notes.json was not created")
		}

		// Read and verify contents
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config file: %v", err)
		}

		var loaded NotesConfig
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("failed to unmarshal saved config: %v", err)
		}

		if loaded.Enabled != config.Enabled {
			t.Error("saved Enabled doesn't match")
		}

		if loaded.MaxExcerptLength != config.MaxExcerptLength {
			t.Error("saved MaxExcerptLength doesn't match")
		}

		if loaded.MaxPrompts != config.MaxPrompts {
			t.Error("saved MaxPrompts doesn't match")
		}

		if loaded.NotesRef != config.NotesRef {
			t.Error("saved NotesRef doesn't match")
		}

		if !reflect.DeepEqual(loaded.ExcludePatterns, config.ExcludePatterns) {
			t.Error("saved ExcludePatterns doesn't match")
		}
	})

	t.Run("overwrite existing config", func(t *testing.T) {
		// Create .claude directory
		claudeDir := filepath.Join(tempDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("failed to create .claude dir: %v", err)
		}

		// Write initial config
		initialConfig := &NotesConfig{Enabled: true}
		if err := SaveNotesConfig(tempDir, initialConfig); err != nil {
			t.Fatalf("failed to save initial config: %v", err)
		}

		// Save new config
		newConfig := &NotesConfig{Enabled: false, MaxPrompts: 20}
		if err := SaveNotesConfig(tempDir, newConfig); err != nil {
			t.Fatalf("failed to save new config: %v", err)
		}

		// Load and verify it was overwritten
		loaded := LoadNotesConfig(tempDir)
		if loaded.Enabled != false {
			t.Error("config was not overwritten")
		}

		if loaded.MaxPrompts != 20 {
			t.Error("config was not fully overwritten")
		}
	})
}

func TestConfigRoundTrip(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-roundtrip-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config with all custom values
	original := &NotesConfig{
		Enabled:           false,
		MaxExcerptLength:  2500,
		MaxPrompts:        15,
		IncludeToolOutput: true,
		NotesRef:          "my-notes",
		ExcludePatterns:   []string{"pattern1", "pattern2", "pattern3"},
		UserEmoji:         "👨‍💻",
		AssistantEmoji:    "🤖",
	}

	// Save it
	if err := SaveNotesConfig(tempDir, original); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load it back
	loaded := LoadNotesConfig(tempDir)

	// Verify all fields match
	if loaded.Enabled != original.Enabled {
		t.Error("Enabled doesn't match after round trip")
	}

	if loaded.MaxExcerptLength != original.MaxExcerptLength {
		t.Error("MaxExcerptLength doesn't match after round trip")
	}

	if loaded.MaxPrompts != original.MaxPrompts {
		t.Error("MaxPrompts doesn't match after round trip")
	}

	if loaded.IncludeToolOutput != original.IncludeToolOutput {
		t.Error("IncludeToolOutput doesn't match after round trip")
	}

	if loaded.NotesRef != original.NotesRef {
		t.Error("NotesRef doesn't match after round trip")
	}

	if !reflect.DeepEqual(loaded.ExcludePatterns, original.ExcludePatterns) {
		t.Error("ExcludePatterns doesn't match after round trip")
	}

	if loaded.UserEmoji != original.UserEmoji {
		t.Error("UserEmoji doesn't match after round trip")
	}

	if loaded.AssistantEmoji != original.AssistantEmoji {
		t.Error("AssistantEmoji doesn't match after round trip")
	}
}
