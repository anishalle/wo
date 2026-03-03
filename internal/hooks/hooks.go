package hooks

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

type Request struct {
	Clean       bool
	Profile     string
	ForceGlobal bool
}

type Plan struct {
	Commands         []string
	ReturnToOriginal bool
}

type Service struct {
	store *db.Store
	cfg   config.Config
}

func New(store *db.Store, cfg config.Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) CommandsForWorkspace(ctx context.Context, ws model.Workspace, req Request) (Plan, error) {
	var plan Plan
	if req.Clean || !s.cfg.Hooks.Enabled {
		return plan, nil
	}
	woCfg, exists, err := config.LoadWorkspaceFile(ws.Path)
	if err != nil {
		return plan, err
	}
	workspaceStartup := []string{}
	if exists {
		workspaceStartup = append(workspaceStartup, woCfg.Enter.Commands...)
	}

	selectedProfile, source, err := s.resolveProfile(req, woCfg)
	if err != nil {
		return plan, err
	}
	if source != "" {
		plan.ReturnToOriginal = !selectedProfile.Chdir
	}

	workspaceCommands := make([]string, 0, len(workspaceStartup)+len(selectedProfile.Commands))
	workspaceCommands = append(workspaceCommands, workspaceStartup...)
	if source == "workspace" {
		workspaceCommands = append(workspaceCommands, selectedProfile.Commands...)
	}

	allowedWorkspaceCommands, err := s.allowWorkspaceCommands(ctx, ws, workspaceCommands)
	if err != nil {
		return plan, err
	}
	plan.Commands = append(plan.Commands, allowedWorkspaceCommands...)
	if source == "global" {
		plan.Commands = append(plan.Commands, selectedProfile.Commands...)
	}
	return plan, nil
}

func (s *Service) resolveProfile(req Request, workspaceCfg model.WorkspaceFileConfig) (model.HookProfile, string, error) {
	var empty model.HookProfile
	if req.Profile == "" {
		return empty, "", nil
	}
	if req.ForceGlobal {
		return s.profileFromGlobal(req.Profile)
	}
	if profile, ok := workspaceCfg.Profiles[req.Profile]; ok {
		if err := validateProfile(req.Profile, profile); err != nil {
			return empty, "", err
		}
		return profile, "workspace", nil
	}
	globalProfile, source, err := s.profileFromGlobal(req.Profile)
	if err != nil {
		return empty, "", err
	}
	return globalProfile, source, nil
}

func (s *Service) profileFromGlobal(profileName string) (model.HookProfile, string, error) {
	var empty model.HookProfile
	globalCfg, globalExists, err := config.LoadGlobalHookFile()
	if err != nil {
		return empty, "", err
	}
	if !globalExists {
		return empty, "", fmt.Errorf("hook profile %q not found", profileName)
	}
	if globalCfg.HasEnter {
		path, pathErr := config.GlobalHookConfigPath()
		if pathErr != nil {
			path = "~/.config/wo/config.wo"
		}
		fmt.Fprintf(os.Stderr, "wo: warning: ignoring [enter] in global hook config %s\n", path)
	}
	profile, ok := globalCfg.Profiles[profileName]
	if !ok {
		return empty, "", fmt.Errorf("hook profile %q not found", profileName)
	}
	if err := validateProfile(profileName, profile); err != nil {
		return empty, "", err
	}
	return profile, "global", nil
}

func validateProfile(name string, profile model.HookProfile) error {
	if len(profile.Commands) == 0 {
		return fmt.Errorf("hook profile %q has no commands", name)
	}
	return nil
}

func (s *Service) allowWorkspaceCommands(ctx context.Context, ws model.Workspace, commands []string) ([]string, error) {
	if len(commands) == 0 {
		return nil, nil
	}
	fingerprint, err := config.WorkspaceFingerprint(ws.Path)
	if err != nil {
		return nil, err
	}
	trust, err := s.store.GetTrust(ctx, ws.ID)
	if err == nil {
		if trust.Decision == db.TrustAllow && trust.Fingerprint == fingerprint {
			return commands, nil
		}
		if trust.Decision == db.TrustDeny && trust.Fingerprint == fingerprint {
			return nil, nil
		}
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	allowed, err := promptTrust(ws.Path)
	if err != nil {
		return nil, err
	}
	decision := db.TrustDeny
	if allowed {
		decision = db.TrustAllow
	}
	if err := s.store.SetTrust(ctx, ws.ID, decision, fingerprint); err != nil {
		return nil, err
	}
	if allowed {
		return commands, nil
	}
	return nil, nil
}

func promptTrust(path string) (bool, error) {
	fmt.Fprintf(os.Stderr, "wo: workspace hook trust for %s\n", path)
	fmt.Fprint(os.Stderr, "Allow workspace hooks for this workspace? (Y/n) ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes", nil
}
