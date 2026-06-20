package models

import "time"

type User struct {
    ID        string    `json:"id"`
    Username  string    `json:"username"`
    IsOnline  bool      `json:"is_online"`
    LastSeen  time.Time `json:"last_seen"`
    CreatedAt time.Time `json:"created_at"`
}

type Message struct {
    ID         string    `json:"id"`
    FromUserID string    `json:"from_user_id"`
    ToUserID   string    `json:"to_user_id"`
    Content    string    `json:"content"`
    CreatedAt  time.Time `json:"created_at"`
}

type FriendRequest struct {
    ID           string    `json:"id"`
    FromUserID   string    `json:"from_user_id"`
    FromUsername string    `json:"from_username"`
    ToUserID     string    `json:"to_user_id"`
    Status       string    `json:"status"`
    CreatedAt    time.Time `json:"created_at"`
}

type WSMessage struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}