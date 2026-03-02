package resolve

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/anishalle/wo/internal/config"
	"github.com/anishalle/wo/internal/db"
	"github.com/anishalle/wo/internal/model"
)

type Service struct {
	store *db.Store
	cfg   config.Config
}

type Match struct {
	Workspace model.Workspace
	Score     float64
	Kind      string
	LastUsed  time.Time
}

type Result struct {
	Matches        []Match
	Stage          string
	Correction     *Match
	TopSuggestions []Match
}

func New(store *db.Store, cfg config.Config) *Service {
	return &Service{store: store, cfg: cfg}
}

func (s *Service) Resolve(ctx context.Context, query string) (Result, error) {
	query = strings.TrimSpace(query)
	var out Result
	all, err := s.store.ListWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	usage, err := s.store.UsageMap(ctx)
	if err != nil {
		return out, err
	}

	aliasMatches, err := s.store.FindByAlias(ctx, query)
	if err != nil {
		return out, err
	}
	if len(aliasMatches) > 0 {
		out.Stage = "alias"
		out.Matches = toMatches(aliasMatches, usage, 1.0, "alias")
		sortMatches(out.Matches)
		return out, nil
	}

	exact := filter(all, func(w model.Workspace) bool {
		return strings.EqualFold(w.RepoName, query)
	})
	if len(exact) > 0 {
		out.Stage = "exact"
		out.Matches = toMatches(exact, usage, 1.0, "exact")
		sortMatches(out.Matches)
		return out, nil
	}

	prefix := filter(all, func(w model.Workspace) bool {
		return strings.HasPrefix(strings.ToLower(w.RepoName), strings.ToLower(query))
	})
	if len(prefix) > 0 {
		out.Stage = "prefix"
		out.Matches = make([]Match, 0, len(prefix))
		for _, ws := range prefix {
			out.Matches = append(out.Matches, Match{
				Workspace: ws,
				Score:     similarity(query, ws.RepoName),
				Kind:      "prefix",
				LastUsed:  usage[ws.ID],
			})
		}
		sortMatches(out.Matches)
		return out, nil
	}

	fuzzy := make([]Match, 0, len(all))
	for _, ws := range all {
		score := similarity(query, ws.RepoName)
		if score < 0.4 {
			continue
		}
		fuzzy = append(fuzzy, Match{
			Workspace: ws,
			Score:     score,
			Kind:      "fuzzy",
			LastUsed:  usage[ws.ID],
		})
	}
	sortMatches(fuzzy)
	if len(fuzzy) > 0 {
		out.Stage = "fuzzy"
		// Never auto-resolve fuzzy matches. Callers must ask the user to confirm.
		out.Matches = nil
		out.TopSuggestions = topN(fuzzy, 8)
		if s.cfg.Correction.Enabled && shouldSuggestCorrection(fuzzy, s.cfg.Correction.MinScore, s.cfg.Correction.MinGap) {
			first := fuzzy[0]
			out.Correction = &first
		}
	}
	return out, nil
}

func shouldSuggestCorrection(matches []Match, minScore, minGap float64) bool {
	if len(matches) == 0 {
		return false
	}
	top := matches[0].Score
	if top < minScore {
		return false
	}
	if len(matches) == 1 {
		return true
	}
	return (top - matches[1].Score) >= minGap
}

func filter(in []model.Workspace, fn func(model.Workspace) bool) []model.Workspace {
	out := make([]model.Workspace, 0, len(in))
	for _, w := range in {
		if fn(w) {
			out = append(out, w)
		}
	}
	return out
}

func toMatches(ws []model.Workspace, usage map[int64]time.Time, score float64, kind string) []Match {
	out := make([]Match, 0, len(ws))
	for _, w := range ws {
		out = append(out, Match{
			Workspace: w,
			Score:     score,
			Kind:      kind,
			LastUsed:  usage[w.ID],
		})
	}
	return out
}

func topN(in []Match, n int) []Match {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func sortMatches(matches []Match) {
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		a, b := matches[i].LastUsed, matches[j].LastUsed
		if !a.Equal(b) {
			return a.After(b)
		}
		if !strings.EqualFold(matches[i].Workspace.RepoName, matches[j].Workspace.RepoName) {
			return strings.ToLower(matches[i].Workspace.RepoName) < strings.ToLower(matches[j].Workspace.RepoName)
		}
		return strings.ToLower(matches[i].Workspace.Path) < strings.ToLower(matches[j].Workspace.Path)
	})
}
