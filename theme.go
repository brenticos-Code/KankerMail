package main

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents the color palette and styling options
type Theme struct {
	Name         string
	Bg           string
	Fg           string
	SidebarBg    string
	SidebarFg    string
	SidebarSelBg string
	SidebarSelFg string
	ListBg       string
	ListFg       string
	ListSelBg    string
	ListSelFg    string
	ListHeaderBg string
	ListHeaderFg string
	ListUnreadFg string
	ViewBg       string
	ViewFg       string
	Accent       string
	BorderActive string
	BorderDim    string
	Success      string
	Warning      string
	Error        string
}

// Global variable for current theme, defaulting to Neon Dark
var CurrentTheme Theme

// AllThemes lists the available built-in themes
var AllThemes = []Theme{
	{
		Name:         "Neon Dark",
		Bg:           "#0f0c1b", // Deep space purple-black
		Fg:           "#e2e0f7", // Soft bright violet-white
		SidebarBg:    "#16122c", // Rich dark violet
		SidebarFg:    "#a59ebd", // Dim purple
		SidebarSelBg: "#ff007f", // Neon hot pink
		SidebarSelFg: "#ffffff", // Pure white
		ListBg:       "#0f0c1b",
		ListFg:       "#d5d0f2",
		ListSelBg:    "#00f0ff", // Bright neon cyan
		ListSelFg:    "#0f0c1b", // Dark contrast
		ListHeaderBg: "#1d183a",
		ListHeaderFg: "#00f0ff", // Neon cyan
		ListUnreadFg: "#ffab00", // Gold warning color for unread
		ViewBg:       "#0b0816",
		ViewFg:       "#e2e0f7",
		Accent:       "#ff007f",
		BorderActive: "#00f0ff",
		BorderDim:    "#2e2654",
		Success:      "#00ff87", // Neon green
		Warning:      "#ffab00",
		Error:        "#ff3366",
	},
	{
		Name:         "Dracula",
		Bg:           "#282a36",
		Fg:           "#f8f8f2",
		SidebarBg:    "#21222c",
		SidebarFg:    "#6272a4", // Comment gray
		SidebarSelBg: "#bd93f9", // Dracula purple
		SidebarSelFg: "#282a36",
		ListBg:       "#282a36",
		ListFg:       "#f8f8f2",
		ListSelBg:    "#44475a", // Selection
		ListSelFg:    "#50fa7b", // Dracula green
		ListHeaderBg: "#21222c",
		ListHeaderFg: "#8be9fd", // Dracula cyan
		ListUnreadFg: "#ff79c6", // Dracula pink
		ViewBg:       "#282a36",
		ViewFg:       "#f8f8f2",
		Accent:       "#bd93f9",
		BorderActive: "#bd93f9",
		BorderDim:    "#44475a",
		Success:      "#50fa7b",
		Warning:      "#ffb86c",
		Error:        "#ff5555",
	},
	{
		Name:         "Catppuccin Macchiato",
		Bg:           "#24273a", // Macchiato Base
		Fg:           "#cad3f5", // Macchiato Text
		SidebarBg:    "#1e2030", // Macchiato Mantle
		SidebarFg:    "#939ab7", // Macchiato Subtext0
		SidebarSelBg: "#c6a0f6", // Macchiato Mauve
		SidebarSelFg: "#1e2030",
		ListBg:       "#24273a",
		ListFg:       "#cad3f5",
		ListSelBg:    "#363a4f", // Macchiato Surface0
		ListSelFg:    "#8bd5ca", // Macchiato Havarti/Teal
		ListHeaderBg: "#1e2030",
		ListHeaderFg: "#b7bdf8", // Macchiato Lavender
		ListUnreadFg: "#f5a97f", // Macchiato Peach
		ViewBg:       "#1e2030",
		ViewFg:       "#cad3f5",
		Accent:       "#c6a0f6",
		BorderActive: "#b7bdf8",
		BorderDim:    "#363a4f",
		Success:      "#a6da95", // Macchiato Green
		Warning:      "#eed49f", // Macchiato Yellow
		Error:        "#ed8796", // Macchiato Red
	},
	{
		Name:         "Nord",
		Bg:           "#2e3440", // Nord0
		Fg:           "#d8dee9", // Nord4
		SidebarBg:    "#242933", // Darker Polar Night
		SidebarFg:    "#4c566a", // Nord3
		SidebarSelBg: "#88c0d0", // Nord8 Frost
		SidebarSelFg: "#2e3440",
		ListBg:       "#2e3440",
		ListFg:       "#d8dee9",
		ListSelBg:    "#3b4252", // Nord1 Selection
		ListSelFg:    "#8fbcbb", // Nord7 Frost
		ListHeaderBg: "#242933",
		ListHeaderFg: "#81a1c1", // Nord9 Frost
		ListUnreadFg: "#ebcb8b", // Nord13 Yellow
		ViewBg:       "#2e3440",
		ViewFg:       "#d8dee9",
		Accent:       "#88c0d0",
		BorderActive: "#81a1c1",
		BorderDim:    "#3b4252",
		Success:      "#a3be8c", // Nord14 Green
		Warning:      "#ebcb8b",
		Error:        "#bf616a", // Nord11 Red
	},
}

func init() {
	// Initialize with default theme
	CurrentTheme = AllThemes[0]
}

// SetTheme sets the active theme by name
func SetTheme(name string) {
	for _, t := range AllThemes {
		if t.Name == name {
			CurrentTheme = t
			break
		}
	}
}

// UIStyles encapsulates dynamic styles derived from the current theme
type UIStyles struct {
	SidebarTitle    lipgloss.Style
	SidebarItem     lipgloss.Style
	SidebarItemSel  lipgloss.Style
	Sidebar         lipgloss.Style
	ListHeader      lipgloss.Style
	ListItem        lipgloss.Style
	ListItemSel     lipgloss.Style
	ListItemUnread  lipgloss.Style
	ListPane        lipgloss.Style
	ViewTitle       lipgloss.Style
	ViewHeaderLabel lipgloss.Style
	ViewHeaderValue lipgloss.Style
	ViewBody        lipgloss.Style
	ViewPane        lipgloss.Style
	PaneActive      lipgloss.Style
	PaneInactive    lipgloss.Style
	StatusMsg       lipgloss.Style
	StatusHelp      lipgloss.Style
	HelpKey         lipgloss.Style
	HelpDesc        lipgloss.Style
	StatusBar       lipgloss.Style
	TitleTag        lipgloss.Style
	ComposerLabel   lipgloss.Style
	ComposerInput   lipgloss.Style
	ComposerBody    lipgloss.Style
	ModalBorder     lipgloss.Style
}

// GetStyles returns Lip Gloss styles corresponding to the current theme
func GetStyles() UIStyles {
	t := CurrentTheme

	return UIStyles{
		SidebarTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true).
			Padding(1, 1).
			MarginBottom(1),

		SidebarItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SidebarFg)).
			PaddingLeft(2),

		SidebarItemSel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SidebarSelFg)).
			Background(lipgloss.Color(t.SidebarSelBg)).
			Bold(true).
			PaddingLeft(2),

		Sidebar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.SidebarBg)).
			Width(22),

		ListHeader: lipgloss.NewStyle().
			Background(lipgloss.Color(t.ListHeaderBg)).
			Foreground(lipgloss.Color(t.ListHeaderFg)).
			Bold(true).
			Padding(0, 1),

		ListItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ListFg)),

		ListItemSel: lipgloss.NewStyle().
			Background(lipgloss.Color(t.ListSelBg)).
			Foreground(lipgloss.Color(t.ListSelFg)).
			Bold(true),

		ListItemUnread: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ListUnreadFg)).
			Bold(true),

		ListPane: lipgloss.NewStyle().
			Background(lipgloss.Color(t.ListBg)),

		ViewTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true).
			PaddingBottom(1),

		ViewHeaderLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SidebarFg)).
			Width(8),

		ViewHeaderValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Fg)).
			Bold(true),

		ViewBody: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.ViewFg)).
			Padding(1, 0),

		ViewPane: lipgloss.NewStyle().
			Background(lipgloss.Color(t.ViewBg)),

		PaneActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderActive)),

		PaneInactive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.BorderDim)),

		StatusMsg: lipgloss.NewStyle().
			Background(lipgloss.Color(t.Accent)).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1),

		StatusHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SidebarFg)),

		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.SidebarFg)),

		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color(t.SidebarBg)).
			Height(1),

		TitleTag: lipgloss.NewStyle().
			Background(lipgloss.Color(t.Accent)).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1),

		ComposerLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Bold(true).
			Width(10),

		ComposerInput: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(t.BorderDim)),

		ComposerBody: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(t.BorderDim)),

		ModalBorder: lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(t.Accent)).
			Padding(1, 2),
	}
}

// GetGlamourStyle returns the Glamour stylesheet name matching the current theme
func GetGlamourStyle() string {
	switch CurrentTheme.Name {
	case "Dracula":
		return "dracula"
	case "Nord":
		return "nord"
	default:
		return "dark"
	}
}
