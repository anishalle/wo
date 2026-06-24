package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds the wo color palette and the prebuilt lipgloss styles used by the
// TUI. Colors are adaptive so they degrade gracefully on light terminals, in
// 16-color terminals, and with NO_COLOR. The palette is gh-inspired.
//
// The palette is intentionally exported so other commands can adopt it later;
// today only the interactive picker consumes it.
type Theme struct {
	// Semantic colors.
	Accent    lipgloss.AdaptiveColor // primary highlight (links, selection)
	AccentAlt lipgloss.AdaptiveColor // secondary highlight (group owners)
	Text      lipgloss.AdaptiveColor // primary foreground
	Muted     lipgloss.AdaptiveColor // secondary foreground (descriptions, hints)
	Faint     lipgloss.AdaptiveColor // rules, separators
	Selection lipgloss.AdaptiveColor // selected row background
	Success   lipgloss.AdaptiveColor
	Warn      lipgloss.AdaptiveColor
	Danger    lipgloss.AdaptiveColor

	// Prebuilt styles (base, without selection background applied).
	Title      lipgloss.Style // the "wo" banner word
	TitleCount lipgloss.Style // " · N repositories"
	Header     lipgloss.Style // owner group header
	Rule       lipgloss.Style // faint horizontal rule
	Bar        lipgloss.Style // selected-row left bar glyph
	Name       lipgloss.Style // repo name, unselected
	NameActive lipgloss.Style // repo name, selected
	Desc       lipgloss.Style // description / path meta
	FooterKey  lipgloss.Style // footer key glyph (e.g. ⏎)
	FooterDesc lipgloss.Style // footer key label
	FilterText lipgloss.Style // filter input text/prompt
}

// DefaultTheme returns the gh-inspired adaptive theme.
func DefaultTheme() Theme {
	t := Theme{
		Accent:    lipgloss.AdaptiveColor{Light: "#0969DA", Dark: "#58A6FF"},
		AccentAlt: lipgloss.AdaptiveColor{Light: "#8250DF", Dark: "#D2A8FF"},
		Text:      lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#E6EDF3"},
		Muted:     lipgloss.AdaptiveColor{Light: "#656D76", Dark: "#8B949E"},
		Faint:     lipgloss.AdaptiveColor{Light: "#D0D7DE", Dark: "#30363D"},
		Selection: lipgloss.AdaptiveColor{Light: "#EAEEF2", Dark: "#21262D"},
		Success:   lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"},
		Warn:      lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"},
		Danger:    lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#F85149"},
	}
	t.Title = lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	t.TitleCount = lipgloss.NewStyle().Foreground(t.Muted)
	t.Header = lipgloss.NewStyle().Bold(true).Foreground(t.AccentAlt)
	t.Rule = lipgloss.NewStyle().Foreground(t.Faint)
	t.Bar = lipgloss.NewStyle().Foreground(t.Accent)
	t.Name = lipgloss.NewStyle().Foreground(t.Text)
	t.NameActive = lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	t.Desc = lipgloss.NewStyle().Foreground(t.Muted)
	t.FooterKey = lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	t.FooterDesc = lipgloss.NewStyle().Foreground(t.Muted)
	t.FilterText = lipgloss.NewStyle().Foreground(t.Accent)
	return t
}
