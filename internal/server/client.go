package server

import (
	"encoding/json"
	"log"
	"terminal-chat/internal/database"
	"terminal-chat/internal/models"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	username string
	db       *database.DB
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, username string, db *database.DB) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		username: username,
		db:       db,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		c.db.SetUserOnline(c.userID, false)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for %s: %v", c.username, err)
			}
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Invalid message from %s: %v", c.username, err)
			continue
		}

		c.handleMessage(&wsMsg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Send any queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(wsMsg *models.WSMessage) {
	// Add sender info
	switch wsMsg.Type {
	case "private_message":
		if payload, ok := wsMsg.Payload.(map[string]interface{}); ok {
			payload["from_user_id"] = c.userID
			payload["from_username"] = c.username

			toUserID, _ := payload["to_user_id"].(string)
			content, _ := payload["content"].(string)

			if toUserID != "" && content != "" {
				// Save to database
				msg, err := c.db.SaveMessage(c.userID, toUserID, content)
				if err != nil {
					log.Printf("Failed to save message: %v", err)
					return
				}
				payload["id"] = msg.ID
				payload["created_at"] = msg.CreatedAt.Format(time.RFC3339)
			}
			wsMsg.Payload = payload
		}

	case "friend_request":
		if payload, ok := wsMsg.Payload.(map[string]interface{}); ok {
			payload["from_user_id"] = c.userID
			payload["from_username"] = c.username
			wsMsg.Payload = payload
		}

	case "friend_accepted":
		if payload, ok := wsMsg.Payload.(map[string]interface{}); ok {
			payload["from_user_id"] = c.userID
			payload["from_username"] = c.username
			wsMsg.Payload = payload
		}

	case "friend_removed":
		if payload, ok := wsMsg.Payload.(map[string]interface{}); ok {
			payload["from_user_id"] = c.userID
			payload["from_username"] = c.username
			wsMsg.Payload = payload
		}
	}

	c.hub.broadcast <- wsMsg
}
