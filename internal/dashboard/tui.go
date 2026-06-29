package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ESC      = "\033["
	BOLD     = "\033[1m"
	DIM      = "\033[2m"
	RESET    = "\033[0m"
	CLEAR    = "\033[2J"
	HOME     = "\033[H"
	HIDE     = "\033[?25l"
	SHOW     = "\033[?25h"

	BLACK   = "\033[30m"
	RED     = "\033[31m"
	GREEN   = "\033[32m"
	YELLOW  = "\033[33m"
	BLUE    = "\033[34m"
	MAGENTA = "\033[35m"
	CYAN    = "\033[36m"
	WHITE   = "\033[37m"

	BG_BLACK   = "\033[40m"
	BG_RED     = "\033[41m"
	BG_GREEN   = "\033[42m"
	BG_YELLOW  = "\033[43m"
	BG_BLUE    = "\033[44m"
	BG_MAGENTA = "\033[45m"
	BG_CYAN    = "\033[46m"
	BG_DARK    = "\033[48;5;236m"

	GRAY   = "\033[90m"
	LGRAY  = "\033[37m"
	ORANGE = "\033[38;5;214m"
	PURPLE = "\033[38;5;141m"
	TEAL   = "\033[38;5;80m"

	BAR_CHAR = "━"
	DOT      = "●"
	ARROW    = "➜"
)

type StatusProvider interface {
	Status() map[string]interface{}
}

type TUI struct {
	mu        sync.Mutex
	provider  StatusProvider
	interval  time.Duration
	running   bool
	stopCh    chan struct{}
	width     int
	height    int
	startTime time.Time
}

func NewTUI(intervalSec int) *TUI {
	width, height := 80, 24
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	if output, err := cmd.Output(); err == nil {
		fmt.Sscanf(string(output), "%d %d", &height, &width)
	}

	return &TUI{
		interval:  time.Duration(intervalSec) * time.Second,
		stopCh:    make(chan struct{}),
		width:     width,
		height:    height,
		startTime: time.Now(),
	}
}

func (t *TUI) SetProvider(p StatusProvider) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.provider = p
}

func (t *TUI) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.mu.Unlock()

	fmt.Print(HIDE + CLEAR)

	go func() {
		for {
			select {
			case <-t.stopCh:
				fmt.Print(SHOW)
				return
			default:
				t.render()
				time.Sleep(t.interval)
			}
		}
	}()

	return nil
}

func (t *TUI) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		close(t.stopCh)
		fmt.Print(SHOW)
		t.running = false
	}
}

func (t *TUI) ensureSize() {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	if output, err := cmd.Output(); err == nil {
		fmt.Sscanf(string(output), "%d %d", &t.height, &t.width)
	}
}

func (t *TUI) render() {
	t.ensureSize()

	t.mu.Lock()
	provider := t.provider
	t.mu.Unlock()

	out := strings.Builder{}
	out.WriteString(HOME)

	if provider == nil {
		out.WriteString(t.center("NYXORA", t.width))
		out.WriteString("\n\n")
		out.WriteString(t.center("initializing...", t.width))
		fmt.Print(out.String())
		return
	}

	status := provider.Status()

	// Header
	header := fmt.Sprintf(" %sNYXORA%s  %sAdaptive Tunnel Orchestrator%s",
		PURPLE+BOLD, RESET, DIM+GRAY, RESET)
	out.WriteString(header)
	out.WriteString("\n")

	// Separator line
	sep := strings.Repeat("━", t.width-1)
	out.WriteString(fmt.Sprintf("%s%s%s\n", DIM+GRAY, sep, RESET))

	// Status bar
	running, _ := status["running"].(bool)
	connected, _ := status["connected"].(bool)
	active, _ := status["active_transport"].(string)
	bestPath, _ := status["best_path"].(string)
	bestScore, _ := status["best_score"].(float64)
	mode, _ := status["server_mode"].(bool)
	allActive, _ := status["all_active"].(bool)
	remoteAddr, _ := status["remote_addr"].(string)

	modeStr := "smart"
	if mode {
		modeStr = "server"
	} else if allActive {
		modeStr = "all-active"
	}

	statusIcon := RED + DOT
	statusLabel := "disconnected"
	if connected {
		statusIcon = GREEN + DOT
		statusLabel = "connected"
	} else if running {
		statusIcon = YELLOW + DOT
		statusLabel = "running"
	}

	uptime := time.Since(t.startTime).Round(time.Second).String()

	out.WriteString(fmt.Sprintf("\n %s %s%s %s  %s %s  %s %s  %s %s\n",
		CYAN+BOLD+"STATUS", RESET,
		statusIcon, statusLabel,
		GREEN+DOT+" active",
		tern(active != "", active, "none"),
		BLUE+DOT+" mode", modeStr,
		MAGENTA+DOT+" up", uptime,
	))

	if remoteAddr != "" {
		out.WriteString(fmt.Sprintf(" %s %s%s\n", GRAY+ARROW, RESET, remoteAddr))
	}

	if bestPath != "" && bestScore > 0 {
		out.WriteString(fmt.Sprintf(" %s%s best: %s%s (%.1f)\n",
			YELLOW, "★", RESET, bestPath, bestScore))
	}

	// Transports table
	transportsRaw, ok := status["transports"].([]interface{})
	if ok && len(transportsRaw) > 0 {
		out.WriteString(fmt.Sprintf("\n %s%sTRANSPORTS%s\n", BOLD, CYAN, RESET))

		var transportList []map[string]interface{}
		for _, tRaw := range transportsRaw {
			if t, ok := tRaw.(map[string]interface{}); ok {
				transportList = append(transportList, t)
			}
		}

		sort.Slice(transportList, func(i, j int) bool {
			si, _ := transportList[i]["score"].(float64)
			sj, _ := transportList[j]["score"].(float64)
			return si > sj
		})

		// Table header
		headerFmt := fmt.Sprintf(" %s%%-12s %%-8s %%-8s %%-10s %%-10s %%-8s %%-6s%s\n",
			DIM+GRAY, RESET)
		out.WriteString(fmt.Sprintf(headerFmt, "NAME", "TYPE", "STATUS", "SCORE", "LATENCY", "LOSS", "BAR"))

		// Separator
		out.WriteString(fmt.Sprintf(" %s%s%s\n", DIM+GRAY, strings.Repeat("─", t.width-3), RESET))

		for _, tr := range transportList {
			name, _ := tr["name"].(string)
			typ, _ := tr["type"].(string)
			stat, _ := tr["status"].(string)
			score, _ := tr["score"].(float64)
			latency, _ := tr["latency"].(float64)
			loss, _ := tr["loss"].(float64)

			statColor := RED
			statIcon := "○"
			switch stat {
			case "active":
				statColor = GREEN
				statIcon = "●"
			case "testing":
				statColor = YELLOW
				statIcon = "◉"
			case "failed":
				statColor = RED
				statIcon = "✕"
			default:
				statColor = GRAY
				statIcon = "○"
			}

			scoreColor := RED
			if score >= 70 {
				scoreColor = GREEN
			} else if score >= 40 {
				scoreColor = YELLOW
			}

			latColor := GREEN
			if latency > 150 {
				latColor = RED
			} else if latency > 80 {
				latColor = YELLOW
			}

			lossColor := GREEN
			if loss > 10 {
				lossColor = RED
			} else if loss > 3 {
				lossColor = YELLOW
			}

			// Score bar
			barLen := 15
			filled := int((score / 100) * float64(barLen))
			if filled > barLen {
				filled = barLen
			}
			bar := strings.Repeat(BAR_CHAR, filled) + strings.Repeat("─", barLen-filled)
			barColor := scoreColor

			activeMarker := ""
			if name == active {
				activeMarker = GREEN + "◀" + RESET + " "
			} else {
				activeMarker = "   "
			}

			row := fmt.Sprintf(" %s%s %-10s%s %s%-7s %s%s%-7s%s %s%5.1f%s %s%7.1fms%s %s%5.1f%%%s %s%s%s\n",
				activeMarker,
				BOLD, name, RESET,
				DIM+typ+RESET,
				statColor, statIcon+" "+stat, RESET,
				scoreColor, score, RESET,
				latColor, latency, RESET,
				lossColor, loss, RESET,
				barColor, bar, RESET,
			)
			out.WriteString(row)
		}
	}

	// Failover status
	failoverRaw, ok := status["failover"].(map[string]interface{})
	if ok && len(failoverRaw) > 0 {
		out.WriteString(fmt.Sprintf("\n %s%sFAILOVER%s\n", BOLD, MAGENTA, RESET))
		var names []string
		for k := range failoverRaw {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			val, _ := failoverRaw[name].(string)
			fColor := GREEN
			switch val {
			case "degraded":
				fColor = YELLOW
			case "down":
				fColor = RED
			}
			out.WriteString(fmt.Sprintf("  %s: %s%s%s\n", name, fColor, val, RESET))
		}
	}

	// Network stats footer
	out.WriteString(fmt.Sprintf("\n %s%s", DIM+GRAY, strings.Repeat("━", t.width-1)))
	out.WriteString(fmt.Sprintf("\n  %snyxora v0.1.0%s  │  %srefresh: %.0fs%s  │  %sctrl+c to exit%s\n",
		PURPLE, RESET,
		DIM, t.interval.Seconds(), RESET,
		GRAY, RESET,
	))

	fmt.Print(out.String())
}

func (t *TUI) center(s string, width int) string {
	padding := (width - len(s)) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat(" ", padding) + s
}

func tern(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}


