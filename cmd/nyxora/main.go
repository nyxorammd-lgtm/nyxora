package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/nyxora/nyxora/internal/agent"
	"github.com/nyxora/nyxora/internal/config"
)

var version = "0.1.0"

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[nyxora] ")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "install":
		cmdInstall()
	case "connect":
		cmdConnect()
	case "disconnect":
		cmdDisconnect()
	case "status":
		cmdStatus()
	case "monitor":
		cmdMonitor()
	case "dashboard":
		cmdDashboard()
	case "daemon":
		cmdDaemon()
	case "version":
		fmt.Printf("nyxora v%s\n", version)
	case "help":
		printUsage()
	default:
		fmt.Printf("unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`NYXORA - Adaptive Tunnel Orchestrator

Usage:
  nyxora install                 Install nyxora and check dependencies
  nyxora connect <remote>        Connect to remote (tests all tunnels, picks best)
  nyxora disconnect              Disconnect all tunnels
  nyxora status                  Show connection status
  nyxora monitor <remote>        Monitor latency/packet loss to remote
  nyxora dashboard               Live terminal dashboard (TUI)
  nyxora daemon                  Run as background agent
  nyxora version                 Show version
  nyxora help                    Show this help

Examples:
  nyxora connect 213.32.69.147
  nyxora dashboard
  nyxora daemon`)
}

func loadConfig() *config.Config {
	cfgPath := os.Getenv("NYXORA_CONFIG")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("warning: %v (using defaults)", err)
		cfg = &config.DefaultConfig
	}
	if v := os.Getenv("NYXORA_ALL_ACTIVE"); v == "true" || v == "1" {
		cfg.AllTunnelsActive = true
	}
	return cfg
}

func cmdInstall() {
	fmt.Println("[nyxora] installing nyxora...")

	for _, dir := range []string{"/etc/nyxora", "/etc/nyxora/tunnels", "/etc/nyxora/cache", "/var/log/nyxora"} {
		os.MkdirAll(dir, 0755)
	}

	cfg := loadConfig()
	cfg.Save("/etc/nyxora/config.json")

	fmt.Println("[nyxora] checking dependencies...")
	for _, dep := range []string{"ping", "wg", "ssh", "curl"} {
		path, err := exec.LookPath(dep)
		if err == nil {
			fmt.Printf("  [OK] %s: %s\n", dep, path)
		} else {
			fmt.Printf("  [WARN] %s not found (install manually if needed)\n", dep)
		}
	}

	fmt.Println()
	fmt.Println("nyxora installed successfully!")
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println("  nyxora connect <remote-ip>")
	fmt.Println("  nyxora dashboard")
}

func cmdConnect() {
	if len(os.Args) < 3 {
		fmt.Println("usage: nyxora connect <remote-ip>")
		os.Exit(1)
	}

	cfg := loadConfig()
	a := agent.New(cfg)
	if err := a.Init(); err != nil {
		log.Fatalf("init: %v", err)
	}

	fmt.Printf("[nyxora] testing all transports to %s...\n", os.Args[2])
	if err := a.Connect(os.Args[2]); err != nil {
		log.Fatalf("connect: %v", err)
	}

	printStatus(a.Status())
}

func cmdDisconnect() {
	cfg := loadConfig()
	a := agent.New(cfg)
	a.Init()
	a.Disconnect()
	fmt.Println("[nyxora] disconnected all transports")
}

func cmdStatus() {
	cfg := loadConfig()
	a := agent.New(cfg)
	a.Init()
	printStatus(a.Status())
}

func cmdMonitor() {
	if len(os.Args) < 3 {
		fmt.Println("usage: nyxora monitor <remote-ip>")
		os.Exit(1)
	}
	cfg := loadConfig()
	a := agent.New(cfg)
	a.Init()
	a.Connect(os.Args[2])
}

func cmdDashboard() {
	cfg := loadConfig()
	a := agent.New(cfg)
	if err := a.Init(); err != nil {
		log.Fatalf("init: %v", err)
	}
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	if err := a.RunDashboard(); err != nil {
		log.Fatalf("dashboard: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	<-sigCh
}

func cmdDaemon() {
	cfg := loadConfig()
	log.Printf("starting nyxora daemon v%s", version)

	a := agent.New(cfg)
	if err := a.Init(); err != nil {
		log.Fatalf("agent init: %v", err)
	}
	if err := a.Start(); err != nil {
		log.Fatalf("agent error: %v", err)
	}
}

func printStatus(status map[string]interface{}) {
	running, _ := status["running"].(bool)
	active, _ := status["active_transport"].(string)
	serverMode, _ := status["server_mode"].(bool)
	allActive, _ := status["all_active"].(bool)
	bestPath, _ := status["best_path"].(string)
	bestScore, _ := status["best_score"].(float64)

	fmt.Println("============= NYXORA STATUS =============")
	fmt.Printf("  Running:       %v\n", running)
	fmt.Printf("  Server Mode:   %v\n", serverMode)
	fmt.Printf("  All Active:    %v\n", allActive)
	fmt.Printf("  Active Tunnel: %s\n", active)
	fmt.Printf("  Best Path:     %s (score: %.1f)\n", bestPath, bestScore)
	fmt.Println("------------------------------------------")
	fmt.Println("  Transports:")

	transportsRaw, ok := status["transports"].([]interface{})
	if !ok || len(transportsRaw) == 0 {
		fmt.Println("  (none)")
		fmt.Println("==========================================")
		return
	}

	for _, tRaw := range transportsRaw {
		t, ok := tRaw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := t["name"].(string)
		typ, _ := t["type"].(string)
		stat, _ := t["status"].(string)
		score, _ := t["score"].(float64)
		latency, _ := t["latency"].(float64)
		loss, _ := t["loss"].(float64)

		icon := "○"
		if stat == "active" {
			icon = "●"
		}

		fmt.Printf("    %s %s (%s)\n", icon, name, typ)
		fmt.Printf("      Score:   %.1f\n", score)
		fmt.Printf("      Latency: %.1f ms\n", latency)
		fmt.Printf("      Loss:    %.1f %%\n", loss)
	}
	fmt.Println("==========================================")
}
