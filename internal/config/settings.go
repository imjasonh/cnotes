package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	Hooks []Hook `json:"hooks"`
}

type Hook struct {
	Events   []string `json:"events"`
	Matchers []string `json:"matchers"`
	Cmds     []string `json:"cmds"`
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
	if path := os.Getenv("CLAUDE_SETTINGS_PATH"); path != "" {
		return path
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	return filepath.Join(home, ".claude", "settings.json")
}

func InstallHooks(binaryPath string) error {
	settingsPath := GetSettingsPath()
	settings, err := LoadSettings(settingsPath)
	if err != nil {
		return err
	}

	hookCmd := binaryPath + " run"
	
	allEvents := []string{
		"pre_tool_use",
		"post_tool_use", 
		"user_prompt_submit",
		"stop",
		"subagent_stop",
		"notification",
		"pre_compact",
	}

	found := false
	for i, hook := range settings.Hooks {
		if len(hook.Cmds) > 0 && hook.Cmds[0] == hookCmd {
			settings.Hooks[i].Events = allEvents
			settings.Hooks[i].Matchers = []string{".*"}
			found = true
			break
		}
	}

	if !found {
		settings.Hooks = append(settings.Hooks, Hook{
			Events:   allEvents,
			Matchers: []string{".*"},
			Cmds:     []string{hookCmd},
		})
	}

	return SaveSettings(settingsPath, settings)
}

func UninstallHooks(binaryPath string) error {
	settingsPath := GetSettingsPath()
	settings, err := LoadSettings(settingsPath)
	if err != nil {
		return err
	}

	hookCmd := binaryPath + " run"
	filtered := make([]Hook, 0)
	
	for _, hook := range settings.Hooks {
		if len(hook.Cmds) == 0 || hook.Cmds[0] != hookCmd {
			filtered = append(filtered, hook)
		}
	}

	settings.Hooks = filtered
	return SaveSettings(settingsPath, settings)
}