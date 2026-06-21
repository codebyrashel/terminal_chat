package main

import (
	"log"
	"net/http"
	"os"
	"terminal-chat/internal/crypto"
	"terminal-chat/internal/database"
	"terminal-chat/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	godotenv.Load()

	// Initialize JWT
	crypto.Init()

	// Connect to database
	db, err := database.New()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	// Create server
	srv := server.NewServer(db)

	// Start WebSocket hub
	go srv.Hub.Run()

	// Register routes
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/api/register", srv.HandleRegister)
	mux.HandleFunc("/api/login", srv.HandleLogin)
	mux.HandleFunc("/api/ws", srv.HandleWebSocket)

	// Friend routes
	mux.HandleFunc("/api/friends", srv.HandleGetFriends)
	mux.HandleFunc("/api/friends/request", srv.HandleSendFriendRequest)
	mux.HandleFunc("/api/friends/requests", srv.HandleGetFriendRequests)
	mux.HandleFunc("/api/friends/accept", srv.HandleAcceptFriendRequest)
	mux.HandleFunc("/api/friends/remove", srv.HandleRemoveFriend)

	// Message routes
	mux.HandleFunc("/api/messages", srv.HandleGetMessages)

	// Search routes
	mux.HandleFunc("/api/users/search", srv.HandleSearchUsers)

	// CORS middleware
	handler := corsMiddleware(mux)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("=================================")
	log.Println("  Terminal Chat Server")
	log.Printf("  Listening on :%s", port)
	log.Println("=================================")
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
