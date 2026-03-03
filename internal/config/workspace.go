package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/anishalle/wo/internal/model"
)

func LoadWorkspaceFile(dir string) (model.WorkspaceFileConfig, bool, error) {
	path := filepath.Join(dir, ".wo")
	return loadHookFile(path)
}

func LoadGlobalHookFile() (model.WorkspaceFileConfig, bool, error) {
	path, err := GlobalHookConfigPath()
	if err != nil {
		return model.WorkspaceFileConfig{}, false, err
	}
	return loadHookFile(path)
}

func loadHookFile(path string) (model.WorkspaceFileConfig, bool, error) {
	cfg := model.WorkspaceFileConfig{
		Profiles: map[string]model.HookProfile{},
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, false, nil
	} else if err != nil {
		return cfg, false, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, true, fmt.Errorf("parse %s: %w", path, err)
	}
	raw := map[string]any{}
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return cfg, true, fmt.Errorf("parse %s: %w", path, err)
	}
	profiles, hasEnter, err := parseProfiles(raw)
	if err != nil {
		return cfg, true, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.Profiles = profiles
	cfg.HasEnter = hasEnter
	if cfg.Enter.Shell == "" {
		cfg.Enter.Shell = "inherit"
	}
	return cfg, true, nil
}

func parseProfiles(raw map[string]any) (map[string]model.HookProfile, bool, error) {
	profiles := map[string]model.HookProfile{}
	hasEnter := false
	for key, value := range raw {
		if strings.EqualFold(key, "enter") {
			hasEnter = true
		}
		if isReservedTopLevelKey(key) {
			continue
		}
		table, ok := value.(map[string]any)
		if !ok {
			continue
		}
		profile, err := parseProfileTable(key, table)
		if err != nil {
			return nil, hasEnter, err
		}
		profiles[key] = profile
	}
	return profiles, hasEnter, nil
}

func parseProfileTable(name string, table map[string]any) (model.HookProfile, error) {
	profile := model.HookProfile{Chdir: true}
	commands := make([]string, 0)
	if rawCommand, ok := table["command"]; ok {
		command, ok := rawCommand.(string)
		if !ok {
			return profile, fmt.Errorf("%s.command must be a string", name)
		}
		command = strings.TrimSpace(command)
		if command != "" {
			commands = append(commands, command)
		}
	}
	if rawCommands, ok := table["commands"]; ok {
		parsed, err := parseProfileCommands(name, rawCommands)
		if err != nil {
			return profile, err
		}
		commands = append(commands, parsed...)
	}
	if rawChdir, ok := table["chdir"]; ok {
		chdir, ok := rawChdir.(bool)
		if !ok {
			return profile, fmt.Errorf("%s.chdir must be a boolean", name)
		}
		profile.Chdir = chdir
	}
	profile.Commands = commands
	return profile, nil
}

func parseProfileCommands(name string, raw any) ([]string, error) {
	switch typed := raw.(type) {
	case []string:
		return trimEmptyCommands(typed), nil
	case []any:
		out := make([]string, 0, len(typed))
		for i, item := range typed {
			cmd, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("%s.commands[%d] must be a string", name, i)
			}
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				out = append(out, cmd)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%s.commands must be an array of strings", name)
	}
}

func trimEmptyCommands(in []string) []string {
	out := make([]string, 0, len(in))
	for _, cmd := range in {
		cmd = strings.TrimSpace(cmd)
		if cmd != "" {
			out = append(out, cmd)
		}
	}
	return out
}

func isReservedTopLevelKey(key string) bool {
	return strings.EqualFold(key, "name") ||
		strings.EqualFold(key, "owner") ||
		strings.EqualFold(key, "enter")
}

func WorkspaceFingerprint(dir string) (string, error) {
	path := filepath.Join(dir, ".wo")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return HashWorkspaceContent(dir, b), nil
}
