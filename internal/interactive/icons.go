package interactive

// transportIcons maps transport names to Unicode glyphs for visual distinction in the TUI.
var transportIcons = map[string]string{
	"wireguard":   "◈",
	"openvpn":     "◉",
	"ssh":         "⌘",
	"shadowsocks": "◎",
	"quic":        "▶",
	"hysteria":    "⚡",
	"frp":         "↻",
	"rathole":     "◌",
	"ipsec":       "◆",
	"backhaul":    "⬡",
	"tcp":         "●",
}

var statusIcons = map[string]string{
	"active":  "●",
	"testing": "◉",
	"failed":  "✗",
	"idle":    "○",
}

var logo = []string{
	"  ███   ██ ██   ██  █████  ██████   █████",
	"  ████  ██ ██   ██ ██   ██ ██   ██ ██   ██",
	"  ██ ██ ██ ██   ██ ██   ██ ██████  ██   ██",
	"  ██  ████ ██   ██ ██   ██ ██   ██ ██   ██",
	"  ██   ███  ██████   █████  ██   ██  █████",
}
