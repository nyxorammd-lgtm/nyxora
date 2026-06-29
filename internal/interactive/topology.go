package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type topologyModel struct {
	quitting bool
	width    int
}

func newTopologyModel() *topologyModel {
	return &topologyModel{}
}

func (m *topologyModel) Init() tea.Cmd {
	return nil
}

func (m *topologyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc", "enter":
			m.quitting = true
			return nil, tea.Quit
		case "t", "T":
			m.quitting = true
			return nil, tea.Quit
		case "1", "2", "3":
			switch msg.String() {
			case "1":
				currentTheme = themes["catppuccin-mocha"]
			case "2":
				currentTheme = themes["tokyo-night"]
			case "3":
				currentTheme = themes["catppuccin-latte"]
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return nil, nil
}

func (m *topologyModel) View() string {
	if m.quitting {
		return "\n"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderGradient("  NYXORA Tunnel Topology", currentTheme.GradientA, currentTheme.GradientB))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render(strings.Repeat("─", 56)))
	b.WriteString("\n\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(currentTheme.Border)).
		Padding(0, 1)

	localBox := box.
		Width(22).
		Render(
			primaryStyle().Render("  Local Server") + "\n" +
				dimStyle().Render("  127.0.0.1") + "\n\n" +
				successStyle().Render("  ● Online"),
		)

	remoteBox := box.
		Width(22).
		Render(
			primaryStyle().Render("  Remote Server") + "\n" +
				dimStyle().Render("  <remote-ip>") + "\n\n" +
				successStyle().Render("  ● Connected"),
		)

	arrow := accentStyle().Render("  ────  ")

	b.WriteString(fmt.Sprintf("  %s%s%s\n", localBox, arrow, remoteBox))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render(strings.Repeat("─", 56)))
	b.WriteString("\n")
	b.WriteString(infoStyle().Bold(true).Render("  Active Tunnels"))
	b.WriteString("\n\n")

	activeStatuses := []struct {
		name   string
		status string
		score  float64
	}{
		{"wireguard", "active", 92.4},
		{"shadowsocks", "active", 87.1},
		{"hysteria", "active", 78.6},
		{"ssh", "active", 71.3},
		{"quic", "testing", 45.2},
		{"frp", "testing", 32.8},
	}

	for _, t := range activeStatuses {
		icon := transportIcons[t.name]
		if icon == "" {
			icon = "●"
		}

		var sIcon, sColor string
		switch t.status {
		case "active":
			sIcon = statusIcons["active"]
			sColor = currentTheme.Success
		case "testing":
			sIcon = statusIcons["testing"]
			sColor = currentTheme.Warning
		default:
			sIcon = statusIcons["idle"]
			sColor = currentTheme.TextDim
		}

		bar := renderGradientBar(t.score, 10, currentTheme.GradientA, currentTheme.GradientB)
		line := fmt.Sprintf("    %s %s %-12s %s %s  %s",
			icon,
			lipgloss.NewStyle().Foreground(lipgloss.Color(sColor)).Render(sIcon),
			lipgloss.NewStyle().Foreground(lipgloss.Color(currentTheme.Text)).Render(t.name),
			dimStyle().Render(fmt.Sprintf("%5.1f", t.score)),
			bar,
			dimStyle().Render(t.status),
		)
		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle().Render(strings.Repeat("─", 56)))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render("  q back  1/2/3 theme"))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render("  \u200Ehttps://t.me/NyxoraCore"))
	b.WriteString("\n")

	return b.String()
}
