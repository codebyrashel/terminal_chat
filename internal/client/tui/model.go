package tui

import (
	"fmt"
	"strings"
	"terminal-chat/internal/client"
	"terminal-chat/internal/models"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	authScreen screen = iota
	chatScreen
)

type authMode int

const (
	loginMode authMode = iota
	registerMode
)

type focusField int

const (
	focusUsername focusField = iota
	focusPassword
)

type UIMessage struct {
	Content string
	Time    time.Time
	IsMine  bool
}

type Chat struct {
	FriendID   string
	FriendName string
	Messages   []UIMessage
	Viewport   viewport.Model
}

type Model struct {
	// Screen
	screen   screen
	authMode authMode
	ready    bool
	width    int
	height   int

	// Auth
	username   textinput.Model
	password   textinput.Model
	focusField focusField
	err        string
	statusMsg  string

	// Chat
	chatInput  textinput.Model
	friends    []models.User
	chats      map[string]*Chat
	selected   int
	activeChat string

	// Dialogs
	showAddFriend  bool
	showRequests   bool
	searchInput    textinput.Model
	friendRequests []models.FriendRequest
	searchResults  []models.User

	// API & WebSocket
	api    *client.APIClient
	ws     *client.WSClient
	user   *models.User
	wsMsgs chan models.WSMessage
}

func NewModel(api *client.APIClient) Model {
	// Username input
	uname := textinput.New()
	uname.Placeholder = "Enter username"
	uname.CharLimit = 32
	uname.Width = 28
	uname.Focus()

	// Password input
	pword := textinput.New()
	pword.Placeholder = "Enter password"
	pword.EchoMode = textinput.EchoPassword
	pword.CharLimit = 64
	pword.Width = 28

	// Chat input
	chat := textinput.New()
	chat.Placeholder = "Type a message... (Enter to send)"
	chat.CharLimit = 1000
	chat.Width = 40

	// Search input
	search := textinput.New()
	search.Placeholder = "Search username..."
	search.CharLimit = 32
	search.Width = 25

	return Model{
		screen:         authScreen,
		authMode:       loginMode,
		username:       uname,
		password:       pword,
		focusField:     focusUsername,
		chatInput:      chat,
		searchInput:    search,
		friends:        make([]models.User, 0),
		chats:          make(map[string]*Chat),
		selected:       -1,
		api:            api,
		friendRequests: make([]models.FriendRequest, 0),
		searchResults:  make([]models.User, 0),
		wsMsgs:         make(chan models.WSMessage, 100),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.EnterAltScreen,
		m.waitForWSMsg(),
	)
}

// waitForWSMsg waits for the next WebSocket message
func (m Model) waitForWSMsg() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.wsMsgs
		if !ok {
			return nil
		}
		return wsMsg(msg)
	}
}

// handleWSMessage processes a WebSocket message
func (m *Model) handleWSMessage(msg models.WSMessage) {
	switch msg.Type {
	case "private_message":
		if p, ok := msg.Payload.(map[string]interface{}); ok {
			fromID, _ := p["from_user_id"].(string)
			fromName, _ := p["from_username"].(string)
			content, _ := p["content"].(string)

			var t time.Time
			if ts, ok := p["created_at"].(string); ok {
				t, _ = time.Parse(time.RFC3339, ts)
			}
			if t.IsZero() {
				t = time.Now()
			}
			// show times in local system timezone
			t = t.Local()

			chat, exists := m.chats[fromID]
			if !exists {
				vp := viewport.New(m.width-26, m.height-8)
				chat = &Chat{FriendID: fromID, FriendName: fromName, Viewport: vp}
				m.chats[fromID] = chat
			}
			chat.Messages = append(chat.Messages, UIMessage{Content: content, Time: t, IsMine: false})
			m.updateViewport(chat)

			if m.activeChat != fromID {
				m.statusMsg = fmt.Sprintf("New message from %s", fromName)
			}
		}

	case "friend_request":
		m.fetchRequests()
		m.statusMsg = "New friend request!"

	case "friend_accepted":
		m.fetchFriends()
		m.fetchRequests()
		m.statusMsg = "Friend request accepted!"

	case "friend_removed":
		if p, ok := msg.Payload.(map[string]interface{}); ok {
			id, _ := p["from_user_id"].(string)
			delete(m.chats, id)
			for i, f := range m.friends {
				if f.ID == id {
					m.friends = append(m.friends[:i], m.friends[i+1:]...)
					break
				}
			}
			if m.activeChat == id {
				m.activeChat = ""
				m.selected = -1
			}
		}
		m.statusMsg = "A friend removed you"
	}
}

func (m *Model) updateViewport(chat *Chat) {
	var sb strings.Builder
	for _, msg := range chat.Messages {
		// header: name and time on top
		timeStr := msg.Time.Format("3:04 PM")
		header := ""
		if msg.IsMine {
			header = fmt.Sprintf("You · %s", timeStr)
		} else {
			header = fmt.Sprintf("%s · %s", chat.FriendName, timeStr)
		}

		// content: message text aligned to left edge
		// render header with smaller style, then content below
		sb.WriteString(HeaderStyle.Render(header))
		sb.WriteString("\n")
		if msg.IsMine {
			sb.WriteString(OutgoingMsgStyle.Render(msg.Content))
		} else {
			sb.WriteString(IncomingMsgStyle.Render(msg.Content))
		}
		sb.WriteString("\n\n")
	}
	chat.Viewport.SetContent(sb.String())
	chat.Viewport.GotoBottom()
}

func (m *Model) fetchFriends() {
	friends, err := m.api.GetFriends()
	if err == nil {
		m.friends = friends
	}
}

func (m *Model) fetchRequests() {
	requests, err := m.api.GetFriendRequests()
	if err == nil {
		m.friendRequests = requests
	}
}

// wsMsg wraps WebSocket messages for Bubble Tea
type wsMsg models.WSMessage
