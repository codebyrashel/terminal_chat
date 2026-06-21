package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"terminal-chat/internal/crypto"
	"terminal-chat/internal/database"
	"terminal-chat/internal/models"

	"github.com/gorilla/websocket"
)

type Server struct {
	Hub      *Hub
	DB       *database.DB
	upgrader websocket.Upgrader
}

func NewServer(db *database.DB) *Server {
	return &Server{
		Hub: NewHub(),
		DB:  db,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// HandleRegister handles user registration
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)

	if input.Username == "" || input.Password == "" {
		s.jsonError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	if len(input.Username) < 3 {
		s.jsonError(w, http.StatusBadRequest, "Username must be at least 3 characters")
		return
	}

	if len(input.Password) < 6 {
		s.jsonError(w, http.StatusBadRequest, "Password must be at least 6 characters")
		return
	}

	// Check if username exists
	existing, _, _ := s.DB.GetUserByUsername(input.Username)
	if existing != nil {
		s.jsonError(w, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	hashedPassword, err := crypto.HashPassword(input.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		s.jsonError(w, http.StatusInternalServerError, "Server error")
		return
	}

	// Create user
	user, err := s.DB.CreateUser(input.Username, hashedPassword)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		s.jsonError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	log.Printf("User registered: %s", user.Username)

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"message":  "Registration successful",
	})
}

// HandleLogin handles user login
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		s.jsonError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	input.Username = strings.TrimSpace(input.Username)
	input.Password = strings.TrimSpace(input.Password)

	if input.Username == "" || input.Password == "" {
		s.jsonError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	// Get user
	user, passwordHash, err := s.DB.GetUserByUsername(input.Username)
	if err != nil {
		s.jsonError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check password
	if !crypto.CheckPassword(input.Password, passwordHash) {
		s.jsonError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate token
	token, err := crypto.GenerateToken(user.ID)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		s.jsonError(w, http.StatusInternalServerError, "Server error")
		return
	}

	log.Printf("User logged in: %s", user.Username)

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"token":    token,
	})
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	userID, err := crypto.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	user, err := s.DB.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Set user online
	s.DB.SetUserOnline(userID, true)

	client := NewClient(s.Hub, conn, user.ID, user.Username, s.DB)
	s.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// HandleGetFriends returns the user's friends list
func (s *Server) HandleGetFriends(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	friends, err := s.DB.GetFriends(userID)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if friends == nil {
		friends = []models.User{}
	}

	s.jsonResponse(w, http.StatusOK, friends)
}

// HandleSendFriendRequest sends a friend request
func (s *Server) HandleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input struct {
		Username string `json:"username"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &input)
	input.Username = strings.TrimSpace(input.Username)

	if input.Username == "" {
		s.jsonError(w, http.StatusBadRequest, "Username required")
		return
	}

	friend, _, err := s.DB.GetUserByUsername(input.Username)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, "User not found")
		return
	}

	if friend.ID == userID {
		s.jsonError(w, http.StatusBadRequest, "Cannot add yourself")
		return
	}

	requestID, err := s.DB.SendFriendRequest(userID, friend.ID)
	if err != nil {
		s.jsonError(w, http.StatusConflict, err.Error())
		return
	}

	log.Printf("Friend request: %s -> %s", userID, friend.ID)

	s.jsonResponse(w, http.StatusCreated, map[string]string{
		"message":    "Friend request sent",
		"request_id": requestID,
	})
}

// HandleGetFriendRequests returns pending friend requests
func (s *Server) HandleGetFriendRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	requests, err := s.DB.GetPendingRequests(userID)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if requests == nil {
		requests = []models.FriendRequest{}
	}

	s.jsonResponse(w, http.StatusOK, requests)
}

// HandleAcceptFriendRequest accepts a friend request
func (s *Server) HandleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input struct {
		RequestID string `json:"request_id"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	if input.RequestID == "" {
		s.jsonError(w, http.StatusBadRequest, "Request ID required")
		return
	}

	// Find the request
	requests, _ := s.DB.GetPendingRequests(userID)
	var fromUserID string
	for _, req := range requests {
		if req.ID == input.RequestID {
			fromUserID = req.FromUserID
			break
		}
	}

	if fromUserID == "" {
		s.jsonError(w, http.StatusNotFound, "Request not found")
		return
	}

	err := s.DB.AcceptFriendRequest(input.RequestID, fromUserID, userID)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Failed to accept request")
		return
	}

	log.Printf("Friend request accepted: %s <-> %s", fromUserID, userID)

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Friend request accepted",
	})
}

// HandleRemoveFriend removes a friendship
func (s *Server) HandleRemoveFriend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input struct {
		FriendID string `json:"friend_id"`
	}

	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &input)

	if input.FriendID == "" {
		s.jsonError(w, http.StatusBadRequest, "Friend ID required")
		return
	}

	err := s.DB.RemoveFriend(userID, input.FriendID)
	if err != nil {
		log.Printf("Failed to remove friend: %v", err)
		s.jsonError(w, http.StatusInternalServerError, "Failed to remove friend")
		return
	}

	log.Printf("Friend removed: user=%s friend=%s", userID, input.FriendID)

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"message": "Friend removed",
	})
}

// HandleGetMessages returns conversation between users
func (s *Server) HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	friendID := r.URL.Query().Get("friend_id")
	if friendID == "" {
		s.jsonError(w, http.StatusBadRequest, "Friend ID required")
		return
	}

	messages, err := s.DB.GetConversation(userID, friendID, 100)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if messages == nil {
		messages = []models.Message{}
	}

	s.jsonResponse(w, http.StatusOK, messages)
}

// HandleSearchUsers searches for users
func (s *Server) HandleSearchUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	userID := s.getUserID(r)
	if userID == "" {
		s.jsonError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.jsonError(w, http.StatusBadRequest, "Search query required")
		return
	}

	users, err := s.DB.SearchUsers(query, userID)
	if err != nil {
		s.jsonError(w, http.StatusInternalServerError, "Database error")
		return
	}

	if users == nil {
		users = []models.User{}
	}

	s.jsonResponse(w, http.StatusOK, users)
}

// Helper: get user ID from Authorization header
func (s *Server) getUserID(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	userID, err := crypto.ValidateToken(token)
	if err != nil {
		return ""
	}

	return userID
}

// Helper: send JSON error
func (s *Server) jsonError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Helper: send JSON response
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
