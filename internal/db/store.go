package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/anishalle/wo/internal/model"
)

type Store struct {
	db *sql.DB
}

type TrustDecision string

const (
	TrustUnknown TrustDecision = ""
	TrustAllow   TrustDecision = "allow"
	TrustDeny    TrustDecision = "deny"
)

type TrustRecord struct {
	WorkspaceID int64
	Path        string
	Decision    TrustDecision
	Fingerprint string
	UpdatedAt   time.Time
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "wo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			repo_name TEXT NOT NULL,
			owner TEXT NOT NULL,
			source TEXT NOT NULL,
			last_seen TEXT NOT NULL,
			has_git INTEGER NOT NULL,
			has_wo INTEGER NOT NULL,
			remote_url TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_workspaces_repo_name ON workspaces(repo_name);`,
		`CREATE INDEX IF NOT EXISTS idx_workspaces_owner ON workspaces(owner);`,
		`CREATE TABLE IF NOT EXISTS aliases (
			alias TEXT NOT NULL,
			workspace_id INTEGER NOT NULL,
			PRIMARY KEY(alias, workspace_id),
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_aliases_alias ON aliases(alias);`,
		`CREATE TABLE IF NOT EXISTS usage (
			workspace_id INTEGER PRIMARY KEY,
			last_used TEXT NOT NULL,
			use_count INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS trust (
			workspace_id INTEGER PRIMARY KEY,
			decision TEXT NOT NULL,
			workspace_fingerprint TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS scan_roots (
			path TEXT PRIMARY KEY,
			depth INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func (s *Store) UpsertWorkspace(ctx context.Context, ws model.Workspace, aliases []string) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces(path, repo_name, owner, source, last_seen, has_git, has_wo, remote_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			repo_name=excluded.repo_name,
			owner=excluded.owner,
			source=excluded.source,
			last_seen=excluded.last_seen,
			has_git=excluded.has_git,
			has_wo=excluded.has_wo,
			remote_url=excluded.remote_url;
	`, ws.Path, ws.RepoName, ws.Owner, ws.Source, now, boolToInt(ws.HasGit), boolToInt(ws.HasWO), ws.RemoteURL)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		id, err = s.workspaceIDByPath(ctx, ws.Path)
		if err != nil {
			return 0, err
		}
	}
	if len(aliases) > 0 {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM aliases WHERE workspace_id = ?`, id); err != nil {
			return 0, err
		}
		for _, alias := range aliases {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO aliases(alias, workspace_id) VALUES(?, ?)`, alias, id); err != nil {
				return 0, err
			}
		}
	}
	return id, nil
}

func (s *Store) workspaceIDByPath(ctx context.Context, path string) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE path = ?`, path).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) SaveScanRoot(ctx context.Context, path string, depth int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO scan_roots(path, depth, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET depth=excluded.depth, updated_at=excluded.updated_at
	`, path, depth, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListWorkspaces(ctx context.Context) ([]model.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, repo_name, owner, source, last_seen, has_git, has_wo, remote_url
		FROM workspaces
		ORDER BY owner COLLATE NOCASE ASC, repo_name COLLATE NOCASE ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

func (s *Store) ListWorkspacesByOwner(ctx context.Context, owner string) ([]model.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, repo_name, owner, source, last_seen, has_git, has_wo, remote_url
		FROM workspaces
		WHERE owner = ?
		ORDER BY repo_name COLLATE NOCASE ASC
	`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

func scanWorkspaces(rows *sql.Rows) ([]model.Workspace, error) {
	var out []model.Workspace
	for rows.Next() {
		var w model.Workspace
		var lastSeen string
		var hasGit, hasWO int
		if err := rows.Scan(&w.ID, &w.Path, &w.RepoName, &w.Owner, &w.Source, &lastSeen, &hasGit, &hasWO, &w.RemoteURL); err != nil {
			return nil, err
		}
		w.HasGit = hasGit == 1
		w.HasWO = hasWO == 1
		if ts, err := time.Parse(time.RFC3339Nano, lastSeen); err == nil {
			w.LastSeen = ts
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) WorkspaceByID(ctx context.Context, id int64) (model.Workspace, error) {
	var w model.Workspace
	var lastSeen string
	var hasGit, hasWO int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, path, repo_name, owner, source, last_seen, has_git, has_wo, remote_url
		FROM workspaces WHERE id = ?
	`, id).Scan(&w.ID, &w.Path, &w.RepoName, &w.Owner, &w.Source, &lastSeen, &hasGit, &hasWO, &w.RemoteURL)
	if err != nil {
		return w, err
	}
	w.HasGit = hasGit == 1
	w.HasWO = hasWO == 1
	if ts, err := time.Parse(time.RFC3339Nano, lastSeen); err == nil {
		w.LastSeen = ts
	}
	return w, nil
}

func (s *Store) FindByRepoName(ctx context.Context, repo string) ([]model.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, repo_name, owner, source, last_seen, has_git, has_wo, remote_url
		FROM workspaces
		WHERE repo_name = ?
		ORDER BY repo_name COLLATE NOCASE ASC
	`, repo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

func (s *Store) FindByAlias(ctx context.Context, alias string) ([]model.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.path, w.repo_name, w.owner, w.source, w.last_seen, w.has_git, w.has_wo, w.remote_url
		FROM aliases a
		JOIN workspaces w ON w.id = a.workspace_id
		WHERE a.alias = ?
		ORDER BY w.repo_name COLLATE NOCASE ASC
	`, alias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

func (s *Store) TouchUsage(ctx context.Context, workspaceID int64) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO usage(workspace_id, last_used, use_count)
		VALUES (?, ?, 1)
		ON CONFLICT(workspace_id) DO UPDATE SET
			last_used=excluded.last_used,
			use_count=use_count+1
	`, workspaceID, now)
	return err
}

func (s *Store) DeleteWorkspaceByID(ctx context.Context, workspaceID int64) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.ExecContext(ctx, `DELETE FROM aliases WHERE workspace_id = ?`, workspaceID); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM usage WHERE workspace_id = ?`, workspaceID); err != nil {
		return false, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM trust WHERE workspace_id = ?`, workspaceID); err != nil {
		return false, err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, workspaceID)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Store) UsageMap(ctx context.Context) (map[int64]time.Time, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT workspace_id, last_used FROM usage`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]time.Time{}
	for rows.Next() {
		var id int64
		var ts string
		if err := rows.Scan(&id, &ts); err != nil {
			return nil, err
		}
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			out[id] = t
		}
	}
	return out, rows.Err()
}

func (s *Store) SetTrust(ctx context.Context, workspaceID int64, decision TrustDecision, fingerprint string) error {
	if decision != TrustAllow && decision != TrustDeny {
		return fmt.Errorf("invalid trust decision: %s", decision)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO trust(workspace_id, decision, workspace_fingerprint, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET
			decision=excluded.decision,
			workspace_fingerprint=excluded.workspace_fingerprint,
			updated_at=excluded.updated_at
	`, workspaceID, decision, fingerprint, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetTrust(ctx context.Context, workspaceID int64) (TrustRecord, error) {
	var rec TrustRecord
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT t.workspace_id, w.path, t.decision, t.workspace_fingerprint, t.updated_at
		FROM trust t
		JOIN workspaces w ON w.id = t.workspace_id
		WHERE t.workspace_id = ?
	`, workspaceID).Scan(&rec.WorkspaceID, &rec.Path, &rec.Decision, &rec.Fingerprint, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return rec, sql.ErrNoRows
	}
	if err != nil {
		return rec, err
	}
	if ts, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		rec.UpdatedAt = ts
	}
	return rec, nil
}

func (s *Store) ListTrust(ctx context.Context) ([]TrustRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.workspace_id, w.path, t.decision, t.workspace_fingerprint, t.updated_at
		FROM trust t
		JOIN workspaces w ON w.id = t.workspace_id
		ORDER BY t.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TrustRecord
	for rows.Next() {
		var rec TrustRecord
		var updatedAt string
		if err := rows.Scan(&rec.WorkspaceID, &rec.Path, &rec.Decision, &rec.Fingerprint, &updatedAt); err != nil {
			return nil, err
		}
		if ts, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
			rec.UpdatedAt = ts
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) ResetTrust(ctx context.Context, workspaceID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM trust WHERE workspace_id = ?`, workspaceID)
	return err
}

func (s *Store) ResetAllTrust(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM trust`)
	return err
}

func (s *Store) DeleteMissingWorkspaces(ctx context.Context, keepPaths []string) (int64, error) {
	if len(keepPaths) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM workspaces`)
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(keepPaths)), ",")
	query := fmt.Sprintf(`DELETE FROM workspaces WHERE path NOT IN (%s)`, placeholders)
	args := make([]any, 0, len(keepPaths))
	for _, p := range keepPaths {
		args = append(args, p)
	}
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
