package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"terminal-chat/internal/models"

	"github.com/gorilla/websocket"
)

type WSClient struct {
	conn   *websocket.Conn
	userID string
	done   chan struct{}
}

func NewWSClient(serverURL, token, userID string) (*WSClient, error) {
	// Convert http:// to ws://
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = fmt.Sprintf("%s/api/ws?token=%s", wsURL, token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket connection failed: %v", err)
	}

	return &WSClient{
		conn:   conn,
		userID: userID,
		done:   make(chan struct{}),
	}, nil
}

// Listen reads messages and sends them to the handler channel
func (c *WSClient) Listen(msgChan chan<- models.WSMessage) {
	defer close(c.done)
	// close the incoming channel when listener exits
	defer func() {
		// Attempt to close receiver channel; ignore panic if already closed.
		defer func() { recover() }()
		close(msgChan)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// ignore unexpected close details
			}
			// connection closed
			return
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			// skip malformed messages
			continue
		}

		select {
		case msgChan <- wsMsg:
			// delivered
		default:
			// channel full, drop
		}
	}
}

// SendMessage sends a message through WebSocket
func (c *WSClient) SendMessage(toUserID, content string) error {
	msg := models.WSMessage{
		Type: "private_message",
		Payload: map[string]interface{}{
			"to_user_id": toUserID,
			"content":    content,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendFriendRequest sends a friend request notification
func (c *WSClient) SendFriendRequest(toUserID string) error {
	msg := models.WSMessage{
		Type: "friend_request",
		Payload: map[string]interface{}{
			"to_user_id": toUserID,
		},
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendFriendAccepted notifies that a friend request was accepted
func (c *WSClient) SendFriendAccepted(toUserID string) error {
	msg := models.WSMessage{
		Type: "friend_accepted",
		Payload: map[string]interface{}{
			"to_user_id": toUserID,
		},
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendFriendRemoved notifies that a friend was removed
func (c *WSClient) SendFriendRemoved(toUserID string) error {
	msg := models.WSMessage{
		Type: "friend_removed",
		Payload: map[string]interface{}{
			"to_user_id": toUserID,
		},
	}

	data, _ := json.Marshal(msg)
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Close closes the WebSocket connection
func (c *WSClient) Close() {
	c.conn.Close()
	<-c.done
}
