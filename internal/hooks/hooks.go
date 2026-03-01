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

type Service struct {
	store *db.Store
	cfg   config.Config
}

func New(store *db.Store, cfg config.Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) CommandsForWorkspace(ctx context.Context, ws model.Workspace, clean bool) ([]string, error) {
	if clean || !s.cfg.Hooks.Enabled {
		return nil, nil
	}
	woCfg, exists, err := config.LoadWorkspaceFile(ws.Path)
	if err != nil {
		return nil, err
	}
	if !exists || len(woCfg.Enter.Commands) == 0 {
		return nil, nil
	}
	fingerprint, err := config.WorkspaceFingerprint(ws.Path)
	if err != nil {
		return nil, err
	}
	trust, err := s.store.GetTrust(ctx, ws.ID)
	if err == nil {
		if trust.Decision == db.TrustAllow && trust.Fingerprint == fingerprint {
			return woCfg.Enter.Commands, nil
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
		return woCfg.Enter.Commands, nil
	}
	return nil, nil
}

func promptTrust(path string) (bool, error) {
	fmt.Fprintf(os.Stderr, "wo: workspace hook trust for %s\n", path)
	fmt.Fprint(os.Stderr, "Allow enter hooks for this workspace? (Y/n) ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes", nil
}
