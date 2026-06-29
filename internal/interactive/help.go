package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpModel struct {
	quitting bool
}

func newHelpModel() *helpModel {
	return &helpModel{}
}

func (m *helpModel) Init() tea.Cmd {
	return nil
}

func (m *helpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc", "enter", "?":
			m.quitting = true
			return nil, tea.Quit
		}
	}
	return nil, nil
}

func (m *helpModel) View() string {
	if m.quitting {
		return "\n"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderGradient("  NYXORA Help", currentTheme.GradientA, currentTheme.GradientB))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render(strings.Repeat("─", 54)))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		items []struct{ key, desc string }
	}{
		{
			"Navigation",
			[]struct{ key, desc string }{
				{"\u2191 \u2193 or j/k", "Move cursor"},
				{"\u23CE or Enter", "Select item"},
				{"Esc", "Go back"},
				{"q", "Quit / Back to menu"},
			},
		},
		{
			"Themes",
			[]struct{ key, desc string }{
				{"1", "Catppuccin Mocha (dark)"},
				{"2", "Tokyo Night (dark)"},
				{"3", "Catppuccin Latte (light)"},
			},
		},
		{
			"Shortcuts",
			[]struct{ key, desc string }{
				{"s", "Toggle status bar"},
				{"?", "This help screen"},
				{"t", "Tunnel topology view"},
				{"Ctrl+C", "Force quit"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(accentStyle().Render("  " + section.title))
		b.WriteString("\n")
		for _, item := range section.items {
			keyStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(currentTheme.Primary)).
				Bold(true).
				Padding(0, 1)
			b.WriteString(fmt.Sprintf("    %s  %s\n", keyStyle.Render(item.key), dimStyle().Render(item.desc)))
		}
		b.WriteString("\n")
	}

	b.WriteString(dimStyle().Render(strings.Repeat("─", 54)))
	b.WriteString("\n")
	b.WriteString(infoStyle().Render("  Press any key to close"))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render("  \u200Ehttps://t.me/NyxoraCore"))
	b.WriteString("\n")

	return b.String()
}
