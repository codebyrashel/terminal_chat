package main

import (
	"fmt"
	"log"
	"os"
	"terminal-chat/internal/client"
	"terminal-chat/internal/client/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	apiClient := client.NewAPIClient(serverURL)
	model := tui.NewModel(apiClient)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(fmt.Sprintf("Error: %v", err))
	}
}
