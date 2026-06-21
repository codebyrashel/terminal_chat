package server

import (
	"encoding/json"
	"log"
	"sync"
	"terminal-chat/internal/models"
)

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *models.WSMessage
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *models.WSMessage, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client
			h.mu.Unlock()
			log.Printf("User connected: %s (%s)", client.username, client.userID)
			h.broadcastStatus(client.userID, true)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("User disconnected: %s (%s)", client.username, client.userID)
			h.broadcastStatus(client.userID, false)

		case message := <-h.broadcast:
			data, _ := json.Marshal(message)
			h.routeMessage(message, data)
		}
	}
}

func (h *Hub) routeMessage(msg *models.WSMessage, data []byte) {
	switch msg.Type {
	case "private_message":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if toUserID, ok := payload["to_user_id"].(string); ok {
				h.sendToUser(toUserID, data)
			}
		}

	case "friend_request":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if toUserID, ok := payload["to_user_id"].(string); ok {
				h.sendToUser(toUserID, data)
			}
		}

	case "friend_accepted":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if toUserID, ok := payload["to_user_id"].(string); ok {
				h.sendToUser(toUserID, data)
			}
		}

	case "friend_removed":
		if payload, ok := msg.Payload.(map[string]interface{}); ok {
			if toUserID, ok := payload["to_user_id"].(string); ok {
				h.sendToUser(toUserID, data)
			}
		}

	case "user_status":
		h.broadcastToAll(data)
	}
}

func (h *Hub) sendToUser(userID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if client, ok := h.clients[userID]; ok {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client.userID)
		}
	}
}

func (h *Hub) broadcastToAll(data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client.userID)
		}
	}
}

func (h *Hub) broadcastStatus(userID string, online bool) {
	msg := &models.WSMessage{
		Type: "user_status",
		Payload: map[string]interface{}{
			"user_id": userID,
			"online":  online,
		},
	}
	h.broadcast <- msg
}

func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}
