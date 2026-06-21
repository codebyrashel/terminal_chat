package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.width < 60 || m.height < 20 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			"Terminal too small. Resize to at least 60x20")
	}

	if m.screen == authScreen {
		return m.renderAuth()
	}
	return m.renderChat()
}

func (m Model) renderAuth() string {
	width := 40

	var s strings.Builder

	// Title
	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Purple).
		Padding(0, 2).
		Width(width).
		Align(lipgloss.Center).
		Render("TERMINAL CHAT"))
	s.WriteString("\n\n")

	// Mode tabs
	loginTab := lipgloss.NewStyle().Padding(0, 2).Render("LOGIN")
	registerTab := lipgloss.NewStyle().Padding(0, 2).Render("REGISTER")
	if m.authMode == loginMode {
		loginTab = lipgloss.NewStyle().Bold(true).Foreground(White).Background(Purple).Padding(0, 2).Render("LOGIN")
	} else {
		registerTab = lipgloss.NewStyle().Bold(true).Foreground(White).Background(Purple).Padding(0, 2).Render("REGISTER")
	}
	s.WriteString(lipgloss.JoinHorizontal(lipgloss.Center, loginTab, " ", registerTab))
	s.WriteString("\n\n")

	// Error / Status
	if m.err != "" {
		s.WriteString(ErrorStyle.Width(width).Render(m.err))
		s.WriteString("\n\n")
	}
	if m.statusMsg != "" {
		s.WriteString(SuccessStyle.Width(width).Render(m.statusMsg))
		s.WriteString("\n\n")
	}

	// Username
	s.WriteString("Username:\n")
	unameStyle := InputStyle
	if m.focusField == focusUsername {
		unameStyle = FocusedInputStyle
	}
	m.username.Width = width - 4
	s.WriteString(unameStyle.Render(m.username.View()))
	s.WriteString("\n\n")

	// Password
	s.WriteString("Password:\n")
	pwordStyle := InputStyle
	if m.focusField == focusPassword {
		pwordStyle = FocusedInputStyle
	}
	m.password.Width = width - 4
	s.WriteString(pwordStyle.Render(m.password.View()))
	s.WriteString("\n\n")

	// Submit button
	btnText := "LOGIN"
	if m.authMode == registerMode {
		btnText = "REGISTER"
	}
	s.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Purple).
		Padding(0, 3).
		Width(width).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("[ Enter to %s ]", btnText)))
	s.WriteString("\n\n")

	// Help
	s.WriteString(HelpStyle.Width(width).Align(lipgloss.Center).Render("Tab: Switch | Ctrl+S: Mode | Ctrl+Q: Quit"))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		AuthBoxStyle.Width(width+4).Render(s.String()))
}

func (m Model) renderChat() string {
	// Title bar
	userInfo := ""
	if m.user != nil {
		userInfo = fmt.Sprintf(" | %s", m.user.Username)
	}
	// Compose title: left-aligned CHAT + username, with nav centered inside the purple bar
	navStr := m.renderNavInline()
	titleContent := fmt.Sprintf("CHAT%s", userInfo)
	totalWidth := m.width - 2

	// center area width is remaining space after titleContent
	centerWidth := totalWidth - lipgloss.Width(titleContent)
	if centerWidth < 0 {
		centerWidth = 0
	}
	centerArea := lipgloss.NewStyle().Width(centerWidth).Align(lipgloss.Center).Render(navStr)
	title := TitleStyle.Copy().Width(totalWidth).Render(lipgloss.JoinHorizontal(lipgloss.Left, titleContent, centerArea))

	// Main content
	sidebarW := 22
	chatW := m.width - sidebarW - 4
	content := lipgloss.JoinHorizontal(lipgloss.Top, m.renderSidebar(sidebarW), m.renderChatArea(chatW))

	// Help bar
	help := HelpStyle.Width(m.width - 2).Render("F1:Add | F2:Requests | Ctrl+N/P:Nav | Ctrl+R:Remove | Ctrl+L:Logout | Ctrl+Q:Quit")

	// preserve previous vertical spacing where nav used to occupy a row
	navPlaceholder := lipgloss.NewStyle().Height(1).Render("")

	// Dialogs
	if m.showAddFriend {
		return lipgloss.JoinVertical(lipgloss.Left, title, navPlaceholder,
			lipgloss.Place(m.width-2, m.height-4, lipgloss.Center, lipgloss.Center, m.renderAddDialog()))
	}
	if m.showRequests {
		return lipgloss.JoinVertical(lipgloss.Left, title, navPlaceholder,
			lipgloss.Place(m.width-2, m.height-4, lipgloss.Center, lipgloss.Center, m.renderRequestsDialog()))
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, navPlaceholder, content, help)
}

func (m Model) renderNav() string {
	chatTab := TabStyle.Render("Chat")
	addTab := TabStyle.Render("Add Friend [F1]")
	reqTab := TabStyle.Render(fmt.Sprintf("Requests [F2] (%d)", len(m.friendRequests)))

	if m.showAddFriend {
		addTab = SelectedTabStyle.Render("Add Friend [F1]")
	} else if m.showRequests {
		reqTab = SelectedTabStyle.Render(fmt.Sprintf("Requests [F2] (%d)", len(m.friendRequests)))
	} else {
		chatTab = SelectedTabStyle.Render("Chat")
	}

	nav := lipgloss.JoinHorizontal(lipgloss.Center, chatTab, addTab, reqTab)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), false, false, true, false).
		BorderForeground(Purple).
		Padding(0, 1).
		Width(m.width - 2).
		Align(lipgloss.Center).
		Render(nav)
}

// renderNavInline returns a compact nav suitable for embedding in the title row
func (m Model) renderNavInline() string {
	chatTab := TabStyle.Render("Chat")
	addTab := TabStyle.Render("Add Friend [F1]")
	reqTab := TabStyle.Render(fmt.Sprintf("Requests [F2] (%d)", len(m.friendRequests)))

	if m.showAddFriend {
		addTab = SelectedTabStyle.Render("Add Friend [F1]")
	} else if m.showRequests {
		reqTab = SelectedTabStyle.Render(fmt.Sprintf("Requests [F2] (%d)", len(m.friendRequests)))
	} else {
		chatTab = SelectedTabStyle.Render("Chat")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, chatTab, addTab, reqTab)
}

func (m Model) renderSidebar(w int) string {
	var s strings.Builder

	s.WriteString(SidebarHeaderStyle.Width(w - 4).Render("FRIENDS"))
	s.WriteString("\n")

	if len(m.friends) == 0 {
		s.WriteString("\n")
		s.WriteString(HelpStyle.Width(w - 6).Align(lipgloss.Center).Render("No friends yet\n\nPress F1 to add"))
	} else {
		for i, friend := range m.friends {
			dot := OfflineDot
			if friend.IsOnline {
				dot = OnlineDot
			}
			name := fmt.Sprintf(" %s %s", dot, truncate(friend.Username, w-8))

			if i == m.selected && friend.ID == m.activeChat {
				s.WriteString(SelectedFriendStyle.Width(w - 6).Render(name))
			} else {
				s.WriteString(FriendItemStyle.Width(w - 6).Render(name))
			}
			s.WriteString("\n")
		}
	}

	return SidebarStyle.Width(w).Height(m.height - 6).Render(s.String())
}

func (m Model) renderChatArea(w int) string {
	if m.activeChat == "" {
		msg := HelpStyle.Width(w - 4).Align(lipgloss.Center).Render("Select a friend to chat\n\nCtrl+N/P: Navigate")
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Gray).
			Width(w).
			Height(m.height - 6).
			Render(lipgloss.Place(w-2, m.height-8, lipgloss.Center, lipgloss.Center, msg))
	}

	chat, ok := m.chats[m.activeChat]
	if !ok {
		return ""
	}

	header := ChatHeaderStyle.Width(w - 2).Render(fmt.Sprintf("Chat with %s", chat.FriendName))

	chat.Viewport.Width = w - 2
	chat.Viewport.Height = m.height - 8
	msgs := chat.Viewport.View()

	m.chatInput.Width = w - 8
	input := ChatInputStyle.Width(w - 2).Render(m.chatInput.View())

	return lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.NewStyle().Height(m.height-8).Render(msgs), input)
}

func (m Model) renderAddDialog() string {
	w := 45
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(White).Render("Add Friend"))
	s.WriteString("\n\n")
	s.WriteString("Search for a user:\n\n")
	m.searchInput.Width = w - 8
	s.WriteString(InputStyle.Width(w - 4).Render(m.searchInput.View()))
	s.WriteString("\n\n")

	if len(m.searchResults) > 0 {
		s.WriteString(fmt.Sprintf("Found %d:\n\n", len(m.searchResults)))
		for i, u := range m.searchResults {
			status := "(offline)"
			if u.IsOnline {
				status = "(online)"
			}
			line := fmt.Sprintf("  %s %s", u.Username, status)
			if i == m.selected {
				line = lipgloss.NewStyle().Foreground(White).Background(Purple).Padding(0, 1).Width(w - 6).Render(fmt.Sprintf("> %s", u.Username))
			}
			s.WriteString(line)
			s.WriteString("\n")
		}
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render("Up/Down: Nav | Ctrl+A: Add | Esc: Close"))
	} else {
		s.WriteString(HelpStyle.Render("Type username and press Enter to search"))
	}

	return DialogStyle.Width(w).Render(s.String())
}

func (m Model) renderRequestsDialog() string {
	w := 45
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(White).Render("Friend Requests"))
	s.WriteString("\n\n")

	if len(m.friendRequests) == 0 {
		s.WriteString(HelpStyle.Render("No pending requests"))
	} else {
		for i, req := range m.friendRequests {
			line := fmt.Sprintf("  From: %s", req.FromUsername)
			if i == m.selected {
				line = lipgloss.NewStyle().Foreground(White).Background(Purple).Padding(0, 1).Width(w - 6).Render(fmt.Sprintf("> %s", req.FromUsername))
			}
			s.WriteString(line)
			s.WriteString("\n")
			s.WriteString(HelpStyle.Render(fmt.Sprintf("    %s", req.CreatedAt.Format("Jan 02 3:04 PM"))))
			s.WriteString("\n\n")
		}
		s.WriteString(HelpStyle.Render("Up/Down: Nav | Enter: Accept | Esc: Close"))
	}

	return DialogStyle.Width(w).Render(s.String())
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
