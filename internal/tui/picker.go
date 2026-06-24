package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/anishalle/wo/internal/model"
)

type pickerMode int

const (
	ModeGrouped pickerMode = iota
	ModeFlat
)

// repoCountLabel renders a count with a correctly pluralized noun.
func repoCountLabel(n int) string {
	if n == 1 {
		return "1 repository"
	}
	return fmt.Sprintf("%d repositories", n)
}

// truncate shortens s to at most max runes, appending an ellipsis when cut. It
// is rune-safe and trims trailing whitespace before adding the ellipsis.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return strings.TrimRight(string(r[:max-1]), " ") + "…"
}

// Layout constants for a rendered row: a 2-cell gutter (left bar + space) and a
// 2-cell gap between the repo-name column and the description.
const (
	rowGutter = 2
	rowGap    = 2
	// nameColMin/Max clamp the auto-sized repo-name column.
	nameColMin = 8
	nameColMax = 28
)

type pickerDelegate struct {
	theme     Theme
	nameWidth int
	showPaths *bool
}

func newPickerDelegate(theme Theme, nameWidth int, showPaths *bool) pickerDelegate {
	if nameWidth < nameColMin {
		nameWidth = nameColMin
	}
	if nameWidth > nameColMax {
		nameWidth = nameColMax
	}
	return pickerDelegate{theme: theme, nameWidth: nameWidth, showPaths: showPaths}
}

func (d pickerDelegate) Height() int                             { return 1 }
func (d pickerDelegate) Spacing() int                            { return 0 }
func (d pickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d pickerDelegate) ShortHelp() []key.Binding                { return []key.Binding{} }
func (d pickerDelegate) FullHelp() [][]key.Binding               { return [][]key.Binding{} }
func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(item)
	if !ok {
		return
	}
	if it.spacer {
		_, _ = io.WriteString(w, "")
		return
	}
	width := m.Width()
	if it.header {
		_, _ = io.WriteString(w, d.renderHeader(it.title, width))
		return
	}

	meta := it.ws.Description
	if d.showPaths != nil && *d.showPaths {
		meta = it.ws.Path
	}
	descMax := width - rowGutter - d.nameWidth - rowGap
	if descMax < 0 {
		descMax = 0
	}
	meta = truncate(meta, descMax)
	name := truncate(it.ws.RepoName, d.nameWidth)

	if index == m.Index() {
		_, _ = io.WriteString(w, d.renderSelected(name, meta, width))
		return
	}

	line := "  " + d.theme.Name.Width(d.nameWidth).Render(name)
	if meta != "" {
		line += strings.Repeat(" ", rowGap) + d.theme.Desc.Render(meta)
	}
	_, _ = io.WriteString(w, line)
}

// renderHeader draws a bold owner name followed by a faint rule filling the row.
func (d pickerDelegate) renderHeader(owner string, width int) string {
	head := "  " + d.theme.Header.Render(owner) + " "
	ruleLen := width - lipgloss.Width(head)
	if ruleLen < 0 {
		ruleLen = 0
	}
	return head + d.theme.Rule.Render(strings.Repeat("─", ruleLen))
}

// renderSelected draws the highlighted row: an accent left bar, bold accent
// name, dim description, all on a subtle background filling the row width. Every
// segment carries the background so there is no gap/bleed across the row.
func (d pickerDelegate) renderSelected(name, meta string, width int) string {
	bg := d.theme.Selection
	fill := func(s string) string { return lipgloss.NewStyle().Background(bg).Render(s) }

	bar := d.theme.Bar.Background(bg).Render("▌")
	nameSeg := d.theme.NameActive.Background(bg).Width(d.nameWidth).Render(name)
	row := bar + fill(" ") + nameSeg
	if meta != "" {
		row += fill(strings.Repeat(" ", rowGap)) + d.theme.Desc.Background(bg).Render(meta)
	}
	if pad := width - lipgloss.Width(row); pad > 0 {
		row += fill(strings.Repeat(" ", pad))
	}
	return row
}

type item struct {
	header bool
	spacer bool
	title  string
	ws     model.Workspace
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.ws.Path }
func (i item) FilterValue() string {
	if i.header {
		return ""
	}
	return i.ws.RepoName + " " + i.ws.Owner + " " + i.ws.Path + " " + i.ws.Description
}

type pickedMsg struct {
	ws model.Workspace
}

type cancelMsg struct{}

type modelPicker struct {
	list      list.Model
	items     []item
	picked    *model.Workspace
	mode      pickerMode
	showPaths *bool
	theme     Theme
	count     int
}

// footerReserve is the number of vertical lines View() draws around the list
// (a blank spacer line plus the footer hint line); the list height is reduced
// by this so nothing is clipped.
const footerReserve = 2

func (m modelPicker) Init() tea.Cmd { return nil }

func (m modelPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	moveDir := 0
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-footerReserve)
	case tea.KeyMsg:
		// While the user is typing a filter query, keystrokes belong to the
		// filter input. Only intercept enter (select the highlighted match) and
		// ctrl+c (quit); everything else falls through to the list so letters
		// like q/s/j/k aren't swallowed by our navigation shortcuts.
		if m.list.FilterState() == list.Filtering {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				if m.selectHighlighted() {
					return m, tea.Quit
				}
				return m, nil
			}
			break
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.selectHighlighted() {
				return m, tea.Quit
			}
			return m, nil
		case "s":
			if m.showPaths != nil {
				*m.showPaths = !*m.showPaths
			}
			return m, nil
		case "j", "down", "ctrl+j", "l", "right", "pgdown", "f", "d", "home", "g", "esc":
			moveDir = 1
		case "k", "up", "ctrl+k", "h", "left", "pgup", "b", "u", "end", "G":
			moveDir = -1
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	// The list recomputes filter matches asynchronously and delivers them via
	// FilterMatchesMsg without moving the cursor — so it would keep pointing at
	// the old index (often the 2nd result). Reset to the top so the best match
	// is highlighted as the user types.
	if _, ok := msg.(list.FilterMatchesMsg); ok {
		m.list.Select(0)
		moveDir = 1
	}
	m.snapToSelectable(moveDir)
	return m, cmd
}

// selectHighlighted records the currently highlighted workspace as the picked
// result. It returns false (no-op) when the cursor is on a header/spacer.
func (m *modelPicker) selectHighlighted() bool {
	selected, ok := m.list.SelectedItem().(item)
	if !ok || selected.header || selected.spacer {
		return false
	}
	ws := selected.ws
	m.picked = &ws
	return true
}

func (m modelPicker) View() string {
	return m.list.View() + "\n" + m.footerView()
}

// footerView renders the key-hint bar: accent key glyphs with dim labels.
func (m modelPicker) footerView() string {
	pathLabel := "path"
	if m.showPaths != nil && *m.showPaths {
		pathLabel = "name"
	}
	hint := func(k, label string) string {
		return m.theme.FooterKey.Render(k) + " " + m.theme.FooterDesc.Render(label)
	}
	parts := []string{
		hint("↑/↓", "move"),
		hint("⏎", "select"),
		hint("/", "filter"),
		hint("s", pathLabel),
		hint("esc", "clear"),
		hint("q", "quit"),
	}
	sep := m.theme.FooterDesc.Render("   ")
	return "  " + strings.Join(parts, sep)
}

func (m *modelPicker) snapToSelectable(preferredDir int) {
	if itemIsSelectable(m.list.SelectedItem()) {
		return
	}
	if preferredDir == 0 {
		preferredDir = 1
	}
	if snapListInDirection(&m.list, preferredDir) {
		return
	}
	_ = snapListInDirection(&m.list, -preferredDir)
}

func snapListInDirection(li *list.Model, dir int) bool {
	if li == nil {
		return false
	}
	items := li.VisibleItems()
	if len(items) == 0 {
		return false
	}
	for i := 0; i < len(items)+2; i++ {
		if itemIsSelectable(li.SelectedItem()) {
			return true
		}
		before := li.Index()
		if dir >= 0 {
			li.CursorDown()
		} else {
			li.CursorUp()
		}
		if li.Index() == before {
			break
		}
	}
	return itemIsSelectable(li.SelectedItem())
}

func itemIsSelectable(it list.Item) bool {
	typed, ok := it.(item)
	if !ok {
		return false
	}
	return !typed.header && !typed.spacer
}

// newPickerModel builds the interactive picker model for the given workspaces.
// In grouped mode workspaces are sorted by owner then repo and owner headers are
// injected. It is separated from PickWorkspace so behavior can be unit-tested
// without running a bubbletea program.
func newPickerModel(title string, workspaces []model.Workspace, grouped bool) modelPicker {
	if grouped {
		sort.SliceStable(workspaces, func(i, j int) bool {
			if !strings.EqualFold(workspaces[i].Owner, workspaces[j].Owner) {
				return strings.ToLower(workspaces[i].Owner) < strings.ToLower(workspaces[j].Owner)
			}
			return strings.ToLower(workspaces[i].RepoName) < strings.ToLower(workspaces[j].RepoName)
		})
	}

	items := make([]list.Item, 0, len(workspaces)+8)
	logicalItems := make([]item, 0, len(workspaces)+8)
	nameWidth := 0
	lastOwner := ""
	for _, ws := range workspaces {
		if n := lipgloss.Width(ws.RepoName); n > nameWidth {
			nameWidth = n
		}
		if grouped && ws.Owner != lastOwner {
			if lastOwner != "" {
				spacer := item{spacer: true}
				items = append(items, spacer)
				logicalItems = append(logicalItems, spacer)
			}
			h := item{header: true, title: ws.Owner}
			items = append(items, h)
			logicalItems = append(logicalItems, h)
			lastOwner = ws.Owner
		}
		it := item{title: ws.RepoName, ws: ws}
		items = append(items, it)
		logicalItems = append(logicalItems, it)
	}
	showPaths := false
	theme := DefaultTheme()
	m := modelPicker{items: logicalItems, showPaths: &showPaths, theme: theme, count: len(workspaces)}
	d := newPickerDelegate(theme, nameWidth, &showPaths)
	li := list.New(items, d, 80, 24)
	li.Title = theme.Title.Render("wo") + theme.TitleCount.Render(fmt.Sprintf(" · %s", repoCountLabel(len(workspaces))))
	li.SetShowHelp(false)
	li.SetShowStatusBar(false)
	li.SetFilteringEnabled(true)
	// Match desired UX:
	// 1) "/" enters filter input mode.
	// 2) First Esc while filtering applies filter and exits input mode.
	// 3) Second Esc clears filter back to full list.
	// Keep Esc from quitting the picker.
	li.KeyMap.AcceptWhileFiltering.SetKeys("enter", "tab", "shift+tab", "ctrl+k", "up", "ctrl+j", "down", "esc")
	li.KeyMap.AcceptWhileFiltering.SetHelp("enter/esc", "apply filter")
	li.KeyMap.CancelWhileFiltering.SetKeys("ctrl+c")
	li.KeyMap.CancelWhileFiltering.SetHelp("ctrl+c", "cancel")
	li.KeyMap.Quit.SetKeys("q")
	li.KeyMap.Quit.SetHelp("q", "quit")
	if grouped && len(logicalItems) > 1 && logicalItems[0].header {
		li.Select(1)
	}
	// Title is pre-styled above; use a plain wrapper so the list's default
	// lavender background isn't applied. The list already inserts one blank
	// line beneath the title.
	li.Styles.Title = lipgloss.NewStyle()
	li.Styles.PaginationStyle = theme.Rule
	li.Styles.FilterPrompt = theme.FilterText
	li.Styles.FilterCursor = theme.FilterText
	li.FilterInput.PromptStyle = theme.FilterText
	li.FilterInput.TextStyle = theme.Name
	li.FilterInput.Cursor.Style = theme.FilterText
	// A static (non-blinking) filter cursor is calmer and avoids a recurring
	// blink timer.
	li.FilterInput.Cursor.SetMode(cursor.CursorStatic)
	m.list = li
	return m
}

func PickWorkspace(title string, workspaces []model.Workspace, grouped bool) (model.Workspace, bool, error) {
	var empty model.Workspace
	if len(workspaces) == 0 {
		return empty, false, nil
	}
	// Detect color from stderr — the stream the UI is rendered to (see
	// tea.WithOutput(os.Stderr) below). Otherwise lipgloss keys its color
	// profile off os.Stdout, which the shell wrapper captures via $(...), so a
	// piped stdout would wrongly disable color on the stderr TTY.
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr))

	m := newPickerModel(title, workspaces, grouped)
	// Render interactive UI on stderr so shell wrappers can safely capture stdout.
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	res, err := p.Run()
	if err != nil {
		return empty, false, err
	}
	finalModel, ok := res.(modelPicker)
	if !ok {
		return empty, false, nil
	}
	if finalModel.picked == nil {
		return empty, false, nil
	}
	return *finalModel.picked, true, nil
}

func HasFZF() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

func PickWithFZF(workspaces []model.Workspace, prompt string) (model.Workspace, bool, error) {
	var empty model.Workspace
	if len(workspaces) == 0 {
		return empty, false, nil
	}
	lines := make([]string, 0, len(workspaces))
	for _, ws := range workspaces {
		lines = append(lines, fmt.Sprintf("%s/%s\t%s", ws.Owner, ws.RepoName, ws.Path))
	}
	cmd := exec.Command("fzf", "--prompt", prompt, "--with-nth=1", "--delimiter=\t")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return empty, false, nil
		}
		return empty, false, err
	}
	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return empty, false, nil
	}
	parts := strings.SplitN(selected, "\t", 2)
	if len(parts) != 2 {
		return empty, false, nil
	}
	path := parts[1]
	for _, ws := range workspaces {
		if ws.Path == path {
			return ws, true, nil
		}
	}
	return empty, false, nil
}
