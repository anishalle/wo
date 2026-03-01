package tui

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

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

type pickerDelegate struct {
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	headerStyle   lipgloss.Style
	dimStyle      lipgloss.Style
}

func newPickerDelegate() pickerDelegate {
	return pickerDelegate{
		selectedStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		normalStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("15")),
		headerStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")).MarginTop(1),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
	}
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
	if it.header {
		line := d.headerStyle.Render(it.title)
		_, _ = io.WriteString(w, line)
		return
	}
	cursor := "  "
	style := d.normalStyle
	if index == m.Index() {
		cursor = "▸ "
		style = d.selectedStyle
	}
	meta := d.dimStyle.Render(fmt.Sprintf("%s · %s", it.ws.Owner, it.ws.Path))
	_, _ = io.WriteString(w, style.Render(cursor+it.ws.RepoName+"  "+meta))
}

type item struct {
	header bool
	title  string
	ws     model.Workspace
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.ws.Path }
func (i item) FilterValue() string {
	if i.header {
		return ""
	}
	return i.ws.RepoName + " " + i.ws.Owner + " " + i.ws.Path
}

type pickedMsg struct {
	ws model.Workspace
}

type cancelMsg struct{}

type modelPicker struct {
	list   list.Model
	items  []item
	picked *model.Workspace
	mode   pickerMode
}

func (m modelPicker) Init() tea.Cmd { return nil }

func (m modelPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			selected, ok := m.list.SelectedItem().(item)
			if !ok || selected.header {
				return m, nil
			}
			copyWs := selected.ws
			m.picked = &copyWs
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m modelPicker) View() string {
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("j/k move  enter select  / filter  q quit")
	return m.list.View() + "\n" + footer
}

func PickWorkspace(title string, workspaces []model.Workspace, grouped bool) (model.Workspace, bool, error) {
	var empty model.Workspace
	if len(workspaces) == 0 {
		return empty, false, nil
	}
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
	lastOwner := ""
	for _, ws := range workspaces {
		if grouped && ws.Owner != lastOwner {
			h := item{header: true, title: ws.Owner}
			items = append(items, h)
			logicalItems = append(logicalItems, h)
			lastOwner = ws.Owner
		}
		it := item{title: ws.RepoName, ws: ws}
		items = append(items, it)
		logicalItems = append(logicalItems, it)
	}
	d := newPickerDelegate()
	li := list.New(items, d, 80, 24)
	li.Title = title
	li.SetShowHelp(false)
	li.SetShowStatusBar(false)
	li.SetFilteringEnabled(true)
	if grouped && len(logicalItems) > 1 && logicalItems[0].header {
		li.Select(1)
	}
	li.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	li.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	m := modelPicker{list: li, items: logicalItems}
	p := tea.NewProgram(m, tea.WithAltScreen())
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
