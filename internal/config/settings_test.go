package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-settings-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(tempDir, "non-existent.json")
		settings, err := LoadSettings(nonExistentPath)

		if err != nil {
			t.Errorf("expected no error for non-existent file, got: %v", err)
		}

		if settings == nil {
			t.Fatal("expected empty settings, got nil")
		}

		if settings.Hooks == nil {
			t.Error("expected empty hooks map, got nil")
		}
	})

	t.Run("valid settings file", func(t *testing.T) {
		settingsPath := filepath.Join(tempDir, "settings.json")

		// Create test settings
		testSettings := &Settings{
			Hooks: map[string][]HookDefinition{
				"PostToolUse": {
					{
						Matcher: ".*",
						Hooks: []HookAction{
							{
								Type:    "command",
								Command: "/usr/bin/cnotes",
							},
						},
					},
				},
			},
		}

		// Write settings
		data, _ := json.MarshalIndent(testSettings, "", "  ")
		if err := os.WriteFile(settingsPath, data, 0644); err != nil {
			t.Fatalf("failed to write settings: %v", err)
		}

		// Load settings
		loaded, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// Verify
		if len(loaded.Hooks) != 1 {
			t.Errorf("expected 1 hook event, got %d", len(loaded.Hooks))
		}

		postToolUseHooks, ok := loaded.Hooks["PostToolUse"]
		if !ok {
			t.Error("PostToolUse hooks not found")
		}

		if len(postToolUseHooks) != 1 {
			t.Errorf("expected 1 PostToolUse hook definition, got %d", len(postToolUseHooks))
		}

		if postToolUseHooks[0].Hooks[0].Command != "/usr/bin/cnotes" {
			t.Errorf("unexpected command: %s", postToolUseHooks[0].Hooks[0].Command)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		settingsPath := filepath.Join(tempDir, "invalid.json")

		// Write invalid JSON
		if err := os.WriteFile(settingsPath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		_, err := LoadSettings(settingsPath)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestSaveSettings(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-save-settings-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("save to new file", func(t *testing.T) {
		settingsPath := filepath.Join(tempDir, "new", "settings.json")

		settings := &Settings{
			Hooks: map[string][]HookDefinition{
				"PreToolUse": {
					{
						Matcher: "^test$",
						Hooks: []HookAction{
							{
								Type:    "command",
								Command: "test-command",
								Timeout: 5000,
							},
						},
					},
				},
			},
		}

		// Save
		if err := SaveSettings(settingsPath, settings); err != nil {
			t.Fatalf("failed to save settings: %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(filepath.Dir(settingsPath)); os.IsNotExist(err) {
			t.Error("directory was not created")
		}

		// Verify file exists
		if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
			t.Error("settings file was not created")
		}

		// Load and verify content
		loaded, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load saved settings: %v", err)
		}

		if len(loaded.Hooks) != 1 {
			t.Error("saved hooks don't match")
		}

		if loaded.Hooks["PreToolUse"][0].Hooks[0].Timeout != 5000 {
			t.Error("timeout was not preserved")
		}
	})
}

func TestGetSettingsPaths(t *testing.T) {
	// Test global path
	globalPath := GetGlobalSettingsPath()
	if globalPath == "" {
		t.Error("expected non-empty global settings path")
	}

	if !filepath.IsAbs(globalPath) {
		t.Error("expected absolute path for global settings")
	}

	// Test local path
	localPath := GetLocalSettingsPath()
	expectedLocal := filepath.Join(".", ".claude", "settings.local.json")
	if localPath != expectedLocal {
		t.Errorf("expected local path %s, got %s", expectedLocal, localPath)
	}

	// Test project path
	projectPath := GetProjectSettingsPath()
	expectedProject := filepath.Join(".", ".claude", "settings.json")
	if projectPath != expectedProject {
		t.Errorf("expected project path %s, got %s", expectedProject, projectPath)
	}

	// Test with environment variable
	testPath := "/custom/path/settings.json"
	os.Setenv("CLAUDE_SETTINGS_PATH", testPath)
	defer os.Unsetenv("CLAUDE_SETTINGS_PATH")

	envPath := GetGlobalSettingsPath()
	if envPath != testPath {
		t.Errorf("expected env path %s, got %s", testPath, envPath)
	}
}

func TestInstallHooksToPath(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-install-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	settingsPath := filepath.Join(tempDir, "settings.json")
	binaryPath := "/usr/bin/cnotes"

	t.Run("install to empty settings", func(t *testing.T) {
		// Install hooks
		if err := InstallHooksToPath(binaryPath, settingsPath); err != nil {
			t.Fatalf("failed to install hooks: %v", err)
		}

		// Load and verify
		settings, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// Should only have PostToolUse
		if len(settings.Hooks) != 1 {
			t.Errorf("expected 1 hook event, got %d", len(settings.Hooks))
		}

		postToolUse, ok := settings.Hooks["PostToolUse"]
		if !ok {
			t.Error("PostToolUse not found")
		}

		if len(postToolUse) != 1 {
			t.Errorf("expected 1 hook definition, got %d", len(postToolUse))
		}

		if postToolUse[0].Hooks[0].Command != binaryPath {
			t.Errorf("expected command %s, got %s", binaryPath, postToolUse[0].Hooks[0].Command)
		}
	})

	t.Run("install with existing hooks", func(t *testing.T) {
		// Create existing settings with a different hook
		existingSettings := &Settings{
			Hooks: map[string][]HookDefinition{
				"PostToolUse": {
					{
						Matcher: ".*",
						Hooks: []HookAction{
							{
								Type:    "command",
								Command: "/other/command",
							},
						},
					},
				},
			},
		}

		if err := SaveSettings(settingsPath, existingSettings); err != nil {
			t.Fatalf("failed to save existing settings: %v", err)
		}

		// Install hooks
		if err := InstallHooksToPath(binaryPath, settingsPath); err != nil {
			t.Fatalf("failed to install hooks: %v", err)
		}

		// Load and verify
		settings, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// Should have both hooks
		postToolUse := settings.Hooks["PostToolUse"]
		if len(postToolUse) != 2 {
			t.Errorf("expected 2 hook definitions, got %d", len(postToolUse))
		}

		// Check both commands exist
		foundOther := false
		foundCnotes := false
		for _, def := range postToolUse {
			for _, hook := range def.Hooks {
				if hook.Command == "/other/command" {
					foundOther = true
				}
				if hook.Command == binaryPath {
					foundCnotes = true
				}
			}
		}

		if !foundOther {
			t.Error("existing hook was removed")
		}

		if !foundCnotes {
			t.Error("cnotes hook was not added")
		}
	})

	t.Run("update existing cnotes hook", func(t *testing.T) {
		// Use a separate settings file for this test
		updateSettingsPath := filepath.Join(tempDir, "update-settings.json")

		// Install with old path
		oldPath := "/old/path/cnotes"
		if err := InstallHooksToPath(oldPath, updateSettingsPath); err != nil {
			t.Fatalf("failed to install initial hooks: %v", err)
		}

		// Install with new path
		newPath := "/new/path/cnotes"
		if err := InstallHooksToPath(newPath, updateSettingsPath); err != nil {
			t.Fatalf("failed to update hooks: %v", err)
		}

		// Load and verify
		settings, err := LoadSettings(updateSettingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// Should not have duplicate hooks
		postToolUse := settings.Hooks["PostToolUse"]
		hookCount := 0
		for _, def := range postToolUse {
			hookCount += len(def.Hooks)
		}

		// Should have exactly 2 (old and new)
		if hookCount != 2 {
			t.Errorf("expected 2 hooks total, got %d", hookCount)
		}
	})
}

func TestUninstallHooksFromPath(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cnotes-uninstall-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	settingsPath := filepath.Join(tempDir, "settings.json")
	binaryPath := "/usr/bin/cnotes"

	t.Run("uninstall from empty settings", func(t *testing.T) {
		// Try to uninstall from non-existent settings
		if err := UninstallHooksFromPath(binaryPath, settingsPath); err != nil {
			t.Fatalf("failed to uninstall: %v", err)
		}

		// Should create empty settings
		settings, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		if len(settings.Hooks) != 0 {
			t.Error("expected empty hooks")
		}
	})

	t.Run("uninstall specific hook", func(t *testing.T) {
		// Create settings with multiple hooks
		settings := &Settings{
			Hooks: map[string][]HookDefinition{
				"PostToolUse": {
					{
						Matcher: ".*",
						Hooks: []HookAction{
							{
								Type:    "command",
								Command: binaryPath,
							},
							{
								Type:    "command",
								Command: "/other/command",
							},
						},
					},
				},
			},
		}

		if err := SaveSettings(settingsPath, settings); err != nil {
			t.Fatalf("failed to save settings: %v", err)
		}

		// Uninstall cnotes
		if err := UninstallHooksFromPath(binaryPath, settingsPath); err != nil {
			t.Fatalf("failed to uninstall: %v", err)
		}

		// Load and verify
		loaded, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// Should still have the other hook
		postToolUse := loaded.Hooks["PostToolUse"]
		if len(postToolUse) != 1 {
			t.Errorf("expected 1 hook definition remaining, got %d", len(postToolUse))
		}

		if postToolUse[0].Hooks[0].Command != "/other/command" {
			t.Error("wrong hook remained")
		}
	})

	t.Run("uninstall removes empty hook definitions", func(t *testing.T) {
		// Create settings with only cnotes hook
		settings := &Settings{
			Hooks: map[string][]HookDefinition{
				"PostToolUse": {
					{
						Matcher: ".*",
						Hooks: []HookAction{
							{
								Type:    "command",
								Command: binaryPath,
							},
						},
					},
				},
			},
		}

		if err := SaveSettings(settingsPath, settings); err != nil {
			t.Fatalf("failed to save settings: %v", err)
		}

		// Uninstall
		if err := UninstallHooksFromPath(binaryPath, settingsPath); err != nil {
			t.Fatalf("failed to uninstall: %v", err)
		}

		// Load and verify
		loaded, err := LoadSettings(settingsPath)
		if err != nil {
			t.Fatalf("failed to load settings: %v", err)
		}

		// PostToolUse should be completely removed
		if _, exists := loaded.Hooks["PostToolUse"]; exists {
			t.Error("expected PostToolUse to be removed entirely")
		}
	})
}
