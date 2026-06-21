package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	Purple      = lipgloss.Color("#7c3aed")
	LightPurple = lipgloss.Color("#a78bfa")
	DarkPurple  = lipgloss.Color("#5b21b6")
	Green       = lipgloss.Color("#10b981")
	Red         = lipgloss.Color("#ef4444")
	Gray        = lipgloss.Color("#6b7280")
	White       = lipgloss.Color("#ffffff")
	Dark        = lipgloss.Color("#1f2937")
	Darker      = lipgloss.Color("#111827")

	// Title bar
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Purple).
			Padding(0, 1)

	// Auth screen
	AuthBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(2, 3)

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Gray).
			Padding(0, 1).
			Width(30)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Purple).
				Padding(0, 1).
				Width(30).
				Background(Darker)

	// Sidebar
	SidebarStyle = lipgloss.NewStyle().
			Width(22).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(0, 1)

	SidebarHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(White).
				Background(Purple).
				Padding(0, 1).
				Align(lipgloss.Center)

	FriendItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedFriendStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Purple).
				Padding(0, 1).
				Margin(1, 0, 1, 0).
				Foreground(White)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(White).
			Background(Purple)

	SelectedTabStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(Dark).
				Background(White).
				Bold(true)

	// Chat area
	ChatHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(LightPurple).
			Padding(0, 1)

	ChatInputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(0, 1)

	// Messages
	MyMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#93c5fd"))

	FriendMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#86efac"))

	// Message rendering
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Bold(true)

	IncomingMsgStyle = lipgloss.NewStyle().
				Align(lipgloss.Left).
				Padding(0, 0).
				Foreground(White)

	OutgoingMsgStyle = lipgloss.NewStyle().
				Align(lipgloss.Left).
				Padding(0, 0).
				Foreground(White)

	TimeStyle = lipgloss.NewStyle().
			Foreground(Gray)

	// Dialog
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Background(Dark).
			Padding(2)

	// Status
	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Red).
			Padding(0, 1)

	OnlineDot  = lipgloss.NewStyle().Foreground(Green).Bold(true).Render("●")
	OfflineDot = lipgloss.NewStyle().Foreground(Gray).Render("○")
)
