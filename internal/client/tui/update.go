package tui

import (
	"fmt"
	"strings"
	"terminal-chat/internal/client"
	"terminal-chat/internal/models"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func connectWS(api *client.APIClient) (*client.WSClient, error) {
	return client.NewWSClient(api.BaseURL, api.Token, api.UserID)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
		for _, chat := range m.chats {
			chat.Viewport.Width = msg.Width - 26
			chat.Viewport.Height = msg.Height - 8
		}
		return m, nil

	case wsMsg:
		// Process WebSocket message - this updates the model directly
		m.handleWSMessage(models.WSMessage(msg))
		// Restart the listener for the next message
		return m, m.waitForWSMsg()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+q":
			if m.ws != nil {
				m.ws.Close()
			}
			return m, tea.Quit
		}

		if m.screen == authScreen {
			return m.updateAuth(msg)
		}
		return m.updateChat(msg)

	case loginMsg:
		m.user = msg.user
		m.screen = chatScreen
		m.chatInput.Focus()

		ws, err := connectWS(m.api)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.ws = ws

		go func() {
			m.ws.Listen(m.wsMsgs)
		}()

		cmds = append(cmds, m.fetchFriendsCmd())
		cmds = append(cmds, m.fetchRequestsCmd())
		return m, tea.Batch(cmds...)

	case registerMsg:
		m.authMode = loginMode
		m.statusMsg = "Registered! Please login."
		m.username.SetValue("")
		m.password.SetValue("")
		m.username.Focus()
		return m, nil

	case friendsMsg:
		m.friends = msg.friends
		if len(m.friends) > 0 && m.activeChat == "" {
			m.selected = 0
			m.activeChat = m.friends[0].ID
			cmds = append(cmds, m.loadMessagesCmd(m.activeChat))
		}
		return m, tea.Batch(cmds...)

	case acceptResultMsg:
		m.friends = msg.friends
		m.friendRequests = msg.requests
		m.statusMsg = msg.message
		// ensure active selection remains valid
		if len(m.friends) > 0 && m.selected == -1 {
			m.selected = 0
			m.activeChat = m.friends[0].ID
			cmds = append(cmds, m.loadMessagesCmd(m.activeChat))
		}
		return m, tea.Batch(cmds...)

	case removeFriendResultMsg:
		m.friends = msg.friends
		delete(m.chats, msg.removedFriendID)
		if m.activeChat == msg.removedFriendID {
			m.activeChat = ""
			m.selected = -1
		}
		return m, nil

	case requestsMsg:
		m.friendRequests = msg.requests
		return m, nil

	case messagesMsg:
		chat, exists := m.chats[msg.friendID]
		if !exists {
			vp := viewport.New(m.width-26, m.height-14)
			chat = &Chat{FriendID: msg.friendID, FriendName: msg.friendName, Viewport: vp}
			m.chats[msg.friendID] = chat
		}
		// convert incoming message times to local timezone
		for i := range msg.messages {
			msg.messages[i].Time = msg.messages[i].Time.Local()
		}
		chat.Messages = msg.messages
		m.updateViewport(chat)
		return m, nil

	case searchMsg:
		m.searchResults = msg.users
		return m, nil

	case statusMsg:
		m.statusMsg = msg.message
		return m, nil

	case errMsg:
		m.err = msg.Error()
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateAuth(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab":
		if m.focusField == focusUsername {
			m.focusField = focusPassword
			m.username.Blur()
			m.password.Focus()
		} else {
			m.focusField = focusUsername
			m.password.Blur()
			m.username.Focus()
		}
		return m, nil

	case "ctrl+s":
		if m.authMode == loginMode {
			m.authMode = registerMode
		} else {
			m.authMode = loginMode
		}
		m.err = ""
		m.statusMsg = ""
		return m, nil

	case "enter":
		return m.handleAuth()
	}

	var cmd tea.Cmd
	if m.focusField == focusUsername {
		m.username, cmd = m.username.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}

func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Always allow top-level tab navigation with F1/F2,
	// even when a dialog is already open.
	switch msg.String() {
	case "f1":
		m.showAddFriend = true
		m.showRequests = false
		m.chatInput.Blur()
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.searchResults = nil
		m.selected = -1
		return m, nil

	case "f2":
		m.showRequests = true
		m.showAddFriend = false
		m.chatInput.Blur()
		m.selected = 0
		return m, m.fetchRequestsCmd()
	}

	if m.showAddFriend {
		return m.updateAddFriendDialog(msg)
	}
	if m.showRequests {
		return m.updateRequestsDialog(msg)
	}

	switch msg.String() {
	case "f1":
		m.showAddFriend = true
		m.showRequests = false
		m.chatInput.Blur()
		m.searchInput.Focus()
		m.searchInput.SetValue("")
		m.searchResults = nil
		m.selected = -1
		return m, nil

	case "f2":
		m.showRequests = true
		m.showAddFriend = false
		m.chatInput.Blur()
		m.selected = 0
		return m, m.fetchRequestsCmd()

	case "esc":
		if m.activeChat != "" {
			m.activeChat = ""
			m.selected = -1
		}
		return m, nil

	case "ctrl+n":
		if len(m.friends) > 0 {
			m.selected = (m.selected + 1) % len(m.friends)
			m.activeChat = m.friends[m.selected].ID
			return m, m.loadMessagesCmd(m.activeChat)
		}

	case "ctrl+p":
		if len(m.friends) > 0 {
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.friends) - 1
			}
			m.activeChat = m.friends[m.selected].ID
			return m, m.loadMessagesCmd(m.activeChat)
		}

	case "ctrl+r":
		if m.activeChat != "" && m.selected >= 0 && m.selected < len(m.friends) {
			friendID := m.friends[m.selected].ID
			friendName := m.friends[m.selected].Username
			return m, m.removeFriendCmd(friendID, friendName)
		}

	case "ctrl+l":
		if m.ws != nil {
			m.ws.Close()
		}
		if m.ws != nil {
			m.ws.Close()
		}
		m.screen = authScreen
		m.user = nil
		m.api.Token = ""
		m.api.UserID = ""
		m.friends = nil
		m.chats = make(map[string]*Chat)
		m.activeChat = ""
		m.selected = -1
		m.wsMsgs = make(chan models.WSMessage, 100)
		m.username.Focus()
		return m, nil

	case "enter":
		if m.activeChat != "" && m.chatInput.Value() != "" {
			content := strings.TrimSpace(m.chatInput.Value())
			if content != "" {
				m.chatInput.SetValue("")
				chat := m.chats[m.activeChat]
				chat.Messages = append(chat.Messages, UIMessage{
					Content: content,
					Time:    time.Now(),
					IsMine:  true,
				})
				m.updateViewport(chat)
				return m, m.sendMessageCmd(m.activeChat, content)
			}
		}
	}

	var cmd tea.Cmd
	m.chatInput, cmd = m.chatInput.Update(msg)
	return m, cmd
}

func (m Model) updateAddFriendDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showAddFriend = false
		m.searchInput.Blur()
		m.chatInput.Focus()
		return m, nil

	case "enter":
		query := strings.TrimSpace(m.searchInput.Value())
		if query != "" {
			m.selected = -1
			return m, m.searchUsersCmd(query)
		}

	case "up", "k":
		if len(m.searchResults) > 0 {
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.searchResults) - 1
			}
		}

	case "down", "j":
		if len(m.searchResults) > 0 {
			m.selected = (m.selected + 1) % len(m.searchResults)
		}

	case "ctrl+a":
		if m.selected >= 0 && m.selected < len(m.searchResults) {
			name := m.searchResults[m.selected].Username
			m.showAddFriend = false
			m.searchInput.Blur()
			m.chatInput.Focus()
			return m, m.sendRequestCmd(name)
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

func (m Model) updateRequestsDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showRequests = false
		m.chatInput.Focus()
		return m, nil

	case "up", "k":
		if len(m.friendRequests) > 0 {
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.friendRequests) - 1
			}
		}

	case "down", "j":
		if len(m.friendRequests) > 0 {
			m.selected = (m.selected + 1) % len(m.friendRequests)
		}

	case "enter":
		if m.selected >= 0 && m.selected < len(m.friendRequests) {
			req := m.friendRequests[m.selected]
			m.showRequests = false
			m.chatInput.Focus()
			return m, m.acceptRequestCmd(req.ID, req.FromUserID)
		}
	}

	return m, nil
}

func (m Model) handleAuth() (tea.Model, tea.Cmd) {
	username := strings.TrimSpace(m.username.Value())
	password := strings.TrimSpace(m.password.Value())

	if username == "" || password == "" {
		m.err = "Username and password required"
		return m, nil
	}

	if m.authMode == loginMode {
		return m, func() tea.Msg {
			user, err := m.api.Login(username, password)
			if err != nil {
				return errMsg{err}
			}
			return loginMsg{user: user}
		}
	} else {
		if len(password) < 6 {
			m.err = "Password must be at least 6 characters"
			return m, nil
		}
		return m, func() tea.Msg {
			_, err := m.api.Register(username, password)
			if err != nil {
				return errMsg{err}
			}
			return registerMsg{}
		}
	}
}

// Commands
func (m Model) sendMessageCmd(friendID, content string) tea.Cmd {
	return func() tea.Msg {
		if m.ws != nil {
			m.ws.SendMessage(friendID, content)
		}
		return nil
	}
}

func (m Model) fetchFriendsCmd() tea.Cmd {
	return func() tea.Msg {
		friends, err := m.api.GetFriends()
		if err != nil {
			return errMsg{err}
		}
		if friends == nil {
			friends = []models.User{}
		}
		return friendsMsg{friends: friends}
	}
}

func (m Model) fetchRequestsCmd() tea.Cmd {
	return func() tea.Msg {
		requests, err := m.api.GetFriendRequests()
		if err != nil {
			return errMsg{err}
		}
		if requests == nil {
			requests = []models.FriendRequest{}
		}
		return requestsMsg{requests: requests}
	}
}

func (m Model) loadMessagesCmd(friendID string) tea.Cmd {
	name := "Unknown"
	for _, f := range m.friends {
		if f.ID == friendID {
			name = f.Username
			break
		}
	}

	return func() tea.Msg {
		msgs, err := m.api.GetMessages(friendID)
		if err != nil {
			return errMsg{err}
		}
		var uiMsgs []UIMessage
		for _, msg := range msgs {
			uiMsgs = append(uiMsgs, UIMessage{
				Content: msg.Content,
				Time:    msg.CreatedAt,
				IsMine:  msg.FromUserID == m.api.UserID,
			})
		}
		return messagesMsg{friendID: friendID, friendName: name, messages: uiMsgs}
	}
}

func (m Model) sendRequestCmd(username string) tea.Cmd {
	return func() tea.Msg {
		err := m.api.SendFriendRequest(username)
		if err != nil {
			return errMsg{err}
		}
		if m.ws != nil {
			// Find the user ID and send WS notification
			for _, u := range m.searchResults {
				if u.Username == username {
					m.ws.SendFriendRequest(u.ID)
					break
				}
			}
		}
		return statusMsg{message: fmt.Sprintf("Request sent to %s!", username)}
	}
}

func (m Model) acceptRequestCmd(requestID, fromUserID string) tea.Cmd {
	return func() tea.Msg {
		err := m.api.AcceptFriendRequest(requestID)
		if err != nil {
			return errMsg{err}
		}
		if m.ws != nil {
			m.ws.SendFriendAccepted(fromUserID)
		}

		// Fetch updated friends and requests on the command goroutine,
		// then return a single message for the main update loop to apply.
		friends, ferr := m.api.GetFriends()
		if ferr != nil || friends == nil {
			friends = []models.User{}
		}
		requests, rerr := m.api.GetFriendRequests()
		if rerr != nil || requests == nil {
			requests = []models.FriendRequest{}
		}
		return acceptResultMsg{friends: friends, requests: requests, message: "Friend request accepted!"}
	}
}

func (m Model) removeFriendCmd(friendID, friendName string) tea.Cmd {
	return func() tea.Msg {
		err := m.api.RemoveFriend(friendID)
		if err != nil {
			return errMsg{err}
		}
		if m.ws != nil {
			m.ws.SendFriendRemoved(friendID)
		}

		// Refresh friends list on success and return it to the main loop.
		friends, ferr := m.api.GetFriends()
		if ferr != nil || friends == nil {
			friends = []models.User{}
		}
		return removeFriendResultMsg{friends: friends, removedFriendID: friendID}
	}
}

func (m Model) searchUsersCmd(query string) tea.Cmd {
	return func() tea.Msg {
		users, err := m.api.SearchUsers(query)
		if err != nil {
			return errMsg{err}
		}
		if users == nil {
			users = []models.User{}
		}
		return searchMsg{users: users}
	}
}

// Message types
type errMsg struct{ error }
type loginMsg struct{ user *models.User }
type registerMsg struct{}
type statusMsg struct{ message string }
type friendsMsg struct{ friends []models.User }
type requestsMsg struct{ requests []models.FriendRequest }
type messagesMsg struct {
	friendID   string
	friendName string
	messages   []UIMessage
}
type searchMsg struct{ users []models.User }
type acceptResultMsg struct {
	friends  []models.User
	requests []models.FriendRequest
	message  string
}

type removeFriendResultMsg struct {
	friends         []models.User
	removedFriendID string
}
