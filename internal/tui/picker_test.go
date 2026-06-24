package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anishalle/wo/internal/model"
)

func TestPickerDelegateShowsDescriptionByDefault(t *testing.T) {
	showPaths := false
	delegate := newPickerDelegate(DefaultTheme(), 12, &showPaths)
	ws := model.Workspace{
		RepoName:    "arlost",
		Owner:       "anishalle",
		Path:        "/Users/ani/workspaces/github.com/anishalle/arlost",
		Description: "A tiny lost-and-found service",
	}
	items := []list.Item{item{title: ws.RepoName, ws: ws}}
	li := list.New(items, delegate, 80, 24)

	var out bytes.Buffer
	delegate.Render(&out, li, 0, items[0])
	rendered := out.String()
	if strings.Contains(rendered, ws.Path) {
		t.Fatalf("expected path to be hidden, got %q", rendered)
	}
	if !strings.Contains(rendered, ws.Description) {
		t.Fatalf("expected description to be visible, got %q", rendered)
	}
}

func TestPickerDelegateHidesOwnerWhenNoDescription(t *testing.T) {
	showPaths := false
	delegate := newPickerDelegate(DefaultTheme(), 12, &showPaths)
	ws := model.Workspace{
		RepoName: "arlost",
		Owner:    "anishalle",
		Path:     "/Users/ani/workspaces/github.com/anishalle/arlost",
	}
	items := []list.Item{item{title: ws.RepoName, ws: ws}}
	li := list.New(items, delegate, 80, 24)

	var out bytes.Buffer
	delegate.Render(&out, li, 0, items[0])
	rendered := out.String()
	if !strings.Contains(rendered, ws.RepoName) {
		t.Fatalf("expected repo name to be visible, got %q", rendered)
	}
	if strings.Contains(rendered, ws.Owner) {
		t.Fatalf("expected owner not to be shown when description is empty, got %q", rendered)
	}
}

func TestPickerDelegateShowsPathWhenEnabled(t *testing.T) {
	showPaths := true
	delegate := newPickerDelegate(DefaultTheme(), 12, &showPaths)
	ws := model.Workspace{
		RepoName:    "arlost",
		Owner:       "anishalle",
		Path:        "/Users/ani/workspaces/github.com/anishalle/arlost",
		Description: "A tiny lost-and-found service",
	}
	items := []list.Item{item{title: ws.RepoName, ws: ws}}
	li := list.New(items, delegate, 80, 24)

	var out bytes.Buffer
	delegate.Render(&out, li, 0, items[0])
	rendered := out.String()
	if !strings.Contains(rendered, ws.Path) {
		t.Fatalf("expected path to be shown, got %q", rendered)
	}
}

func TestRenderHeaderHasOwnerAndRule(t *testing.T) {
	d := newPickerDelegate(DefaultTheme(), 12, nil)
	out := d.renderHeader("anishalle", 40)
	if !strings.Contains(out, "anishalle") {
		t.Fatalf("expected owner in header, got %q", out)
	}
	if !strings.Contains(out, "─") {
		t.Fatalf("expected a rule in header, got %q", out)
	}
}

func TestRenderSelectedHasBarAndFillsWidth(t *testing.T) {
	d := newPickerDelegate(DefaultTheme(), 8, nil)
	out := d.renderSelected("wo", "workspace mgmt", 50)
	if !strings.Contains(out, "▌") {
		t.Fatalf("expected accent bar in selected row, got %q", out)
	}
	if !strings.Contains(out, "wo") || !strings.Contains(out, "workspace mgmt") {
		t.Fatalf("expected name and description in selected row, got %q", out)
	}
	// With color disabled in tests the row is plain text; it should be padded to
	// the full width so the selection background spans the row.
	if w := lipgloss.Width(out); w != 50 {
		t.Fatalf("expected selected row width 50, got %d (%q)", w, out)
	}
}

func TestFooterViewHasHints(t *testing.T) {
	showPaths := false
	m := modelPicker{theme: DefaultTheme(), showPaths: &showPaths}
	out := m.footerView()
	for _, want := range []string{"move", "select", "filter", "path", "quit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected footer to contain %q, got %q", want, out)
		}
	}
	*m.showPaths = true
	if !strings.Contains(m.footerView(), "name") {
		t.Fatalf("expected footer toggle label to flip to 'name' when paths shown")
	}
}

// drive applies a message and then executes the resulting commands (one level
// of tea.Batch deep), feeding their messages back into the model. This lets the
// list's asynchronous filtering (FilterMatchesMsg) actually run in tests.
func drive(m tea.Model, msg tea.Msg) tea.Model {
	m, cmd := m.Update(msg)
	for i := 0; cmd != nil && i < 64; i++ {
		out := cmd()
		cmd = nil
		var msgs []tea.Msg
		if batch, ok := out.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					msgs = append(msgs, c())
				}
			}
		} else if out != nil {
			msgs = []tea.Msg{out}
		}
		for _, mm := range msgs {
			if mm == nil {
				continue
			}
			if _, quit := mm.(tea.QuitMsg); quit {
				continue
			}
			var c tea.Cmd
			m, c = m.Update(mm)
			if c != nil {
				cmd = c
			}
		}
	}
	return m
}

func typeQuery(m tea.Model, query string) tea.Model {
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range query {
		m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func TestFilterHighlightsFirstMatch(t *testing.T) {
	wss := []model.Workspace{
		{RepoName: "armada", Owner: "anishalle"},
		{RepoName: "arlost", Owner: "anishalle"},
		{RepoName: "harp", Owner: "hackutd"},
	}
	var m tea.Model = newPickerModel("wo", wss, true)
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = typeQuery(m, "ar")

	mp := m.(modelPicker)
	sel, ok := mp.list.SelectedItem().(item)
	if !ok {
		t.Fatalf("no item selected while filtering")
	}
	if sel.header {
		t.Fatalf("a header is highlighted while filtering")
	}
	visible := mp.list.VisibleItems()
	if len(visible) == 0 {
		t.Fatalf("expected filter matches")
	}
	first, _ := visible[0].(item)
	if sel.ws.RepoName != first.ws.RepoName {
		t.Fatalf("expected first match %q highlighted, got %q", first.ws.RepoName, sel.ws.RepoName)
	}
}

func TestFilterKeysNotSwallowed(t *testing.T) {
	wss := []model.Workspace{{RepoName: "squash", Owner: "anishalle"}}
	var m tea.Model = newPickerModel("wo", wss, true)
	m = drive(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	// Typing letters that are also navigation/quit shortcuts must reach the
	// filter input rather than being intercepted.
	m = typeQuery(m, "squash")
	mp := m.(modelPicker)
	if got := mp.list.FilterInput.Value(); got != "squash" {
		t.Fatalf("expected filter input %q, got %q", "squash", got)
	}
	if mp.picked != nil {
		t.Fatalf("typing 'q' while filtering must not quit/select")
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 60, "short"},
		{"exactly-five", 12, "exactly-five"},
		{"this is a longer description than allowed", 10, "this is a…"},
		{"trailing space cut here", 6, "trail…"},
		{"☃☃☃☃☃", 3, "☃☃…"},
	}
	for _, c := range cases {
		got := truncate(c.in, c.max)
		if got != c.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
		if r := []rune(got); len(r) > c.max && c.max > 0 {
			t.Errorf("truncate(%q, %d) = %q exceeds max runes %d", c.in, c.max, got, c.max)
		}
	}
}
