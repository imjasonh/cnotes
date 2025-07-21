package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	Hooks map[string][]HookDefinition `json:"hooks"`
}

type HookDefinition struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookAction `json:"hooks"`
}

type HookAction struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Settings{}, nil
		}
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}

	return &settings, nil
}

func SaveSettings(path string, settings *Settings) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

func GetSettingsPath() string {
	return GetGlobalSettingsPath()
}

func GetGlobalSettingsPath() string {
	if path := os.Getenv("CLAUDE_SETTINGS_PATH"); path != "" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".claude", "settings.json")
}

func GetLocalSettingsPath() string {
	return filepath.Join(".", ".claude", "settings.local.json")
}

func GetProjectSettingsPath() string {
	return filepath.Join(".", ".claude", "settings.json")
}

func InstallHooks(binaryPath string) error {
	return InstallHooksToPath(binaryPath, GetSettingsPath())
}

func InstallHooksToPath(binaryPath, settingsPath string) error {
	settings, err := LoadSettings(settingsPath)
	if err != nil {
		return err
	}

	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]HookDefinition)
	}

	hookAction := HookAction{
		Type:    "command",
		Command: binaryPath,
	}

	hookDef := HookDefinition{
		Matcher: ".*",
		Hooks:   []HookAction{hookAction},
	}

	// Map our event names to Claude's event names
	eventMap := map[string]string{
		"pre_tool_use":       "PreToolUse",
		"post_tool_use":      "PostToolUse",
		"user_prompt_submit": "UserPromptSubmit",
		"stop":               "Stop",
		"subagent_stop":      "SubagentStop",
		"notification":       "Notification",
		"pre_compact":        "PreCompact",
	}

	for _, claudeEvent := range eventMap {
		// Check if our hook is already installed
		found := false
		for i, def := range settings.Hooks[claudeEvent] {
			for j, action := range def.Hooks {
				if action.Command == binaryPath {
					// Update existing hook
					settings.Hooks[claudeEvent][i].Hooks[j] = hookAction
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			// Add new hook
			settings.Hooks[claudeEvent] = append(settings.Hooks[claudeEvent], hookDef)
		}
	}

	return SaveSettings(settingsPath, settings)
}

func UninstallHooks(binaryPath string) error {
	return UninstallHooksFromPath(binaryPath, GetSettingsPath())
}

func UninstallHooksFromPath(binaryPath, settingsPath string) error {
	settings, err := LoadSettings(settingsPath)
	if err != nil {
		return err
	}

	if settings.Hooks == nil {
		return nil
	}

	// Remove our hook from all events
	for eventName, hookDefs := range settings.Hooks {
		newDefs := make([]HookDefinition, 0)

		for _, def := range hookDefs {
			newActions := make([]HookAction, 0)
			for _, action := range def.Hooks {
				if action.Command != binaryPath {
					newActions = append(newActions, action)
				}
			}

			// Only keep the definition if it still has actions
			if len(newActions) > 0 {
				def.Hooks = newActions
				newDefs = append(newDefs, def)
			}
		}

		if len(newDefs) > 0 {
			settings.Hooks[eventName] = newDefs
		} else {
			delete(settings.Hooks, eventName)
		}
	}

	return SaveSettings(settingsPath, settings)
}
