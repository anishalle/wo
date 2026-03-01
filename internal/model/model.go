package model

import "time"

// Workspace is a normalized workspace record used throughout wo.
type Workspace struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	RepoName  string    `json:"repo_name"`
	Owner     string    `json:"owner"`
	Source    string    `json:"source"`
	LastSeen  time.Time `json:"last_seen"`
	HasGit    bool      `json:"has_git"`
	HasWO     bool      `json:"has_wo"`
	RemoteURL string    `json:"remote_url"`
}

type WorkspaceFileConfig struct {
	Name  string      `toml:"name"`
	Owner string      `toml:"owner"`
	Enter EnterConfig `toml:"enter"`
}

type EnterConfig struct {
	Commands []string `toml:"commands"`
	Shell    string   `toml:"shell"`
}

type ResolveStatus string

const (
	ResolveOK         ResolveStatus = "ok"
	ResolveNoMatch    ResolveStatus = "no_match"
	ResolveNeedsPick  ResolveStatus = "needs_confirmation"
	ResolveError      ResolveStatus = "error"
	ResolveUserCancel ResolveStatus = "cancelled"
)

type ResolveCandidate struct {
	ID       int64   `json:"id"`
	Path     string  `json:"path"`
	RepoName string  `json:"repo_name"`
	Owner    string  `json:"owner"`
	Score    float64 `json:"score"`
}

type ResolveResponse struct {
	Status            ResolveStatus      `json:"status"`
	Path              string             `json:"path,omitempty"`
	Message           string             `json:"message,omitempty"`
	HookCommands      []string           `json:"hook_commands,omitempty"`
	NeedsConfirmation bool               `json:"needs_confirmation"`
	Candidates        []ResolveCandidate `json:"candidates,omitempty"`
	ExitCode          int                `json:"exit_code"`
}

const (
	ExitOK       = 0
	ExitNoMatch  = 1
	ExitError    = 2
	ExitCanceled = 130
)
