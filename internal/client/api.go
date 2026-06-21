package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"terminal-chat/internal/models"
)

type APIClient struct {
	BaseURL string
	Token   string
	UserID  string
	http    *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *APIClient) Register(username, password string) (*models.User, error) {
	data := map[string]string{"username": username, "password": password}
	resp, err := c.post("/api/register", data, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusCreated {
		if msg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf(msg)
		}
		return nil, fmt.Errorf("registration failed")
	}

	return &models.User{
		ID:       result["id"].(string),
		Username: result["username"].(string),
	}, nil
}

func (c *APIClient) Login(username, password string) (*models.User, error) {
	data := map[string]string{"username": username, "password": password}
	resp, err := c.post("/api/login", data, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		if msg, ok := result["error"].(string); ok {
			return nil, fmt.Errorf(msg)
		}
		return nil, fmt.Errorf("login failed")
	}

	c.Token = result["token"].(string)
	c.UserID = result["id"].(string)

	return &models.User{
		ID:       c.UserID,
		Username: result["username"].(string),
	}, nil
}

func (c *APIClient) GetFriends() ([]models.User, error) {
	resp, err := c.get("/api/friends")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var friends []models.User
	json.NewDecoder(resp.Body).Decode(&friends)
	if friends == nil {
		friends = []models.User{}
	}
	return friends, nil
}

func (c *APIClient) GetMessages(friendID string) ([]models.Message, error) {
	path := fmt.Sprintf("/api/messages?friend_id=%s", url.QueryEscape(friendID))
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var messages []models.Message
	json.NewDecoder(resp.Body).Decode(&messages)
	if messages == nil {
		messages = []models.Message{}
	}
	return messages, nil
}

func (c *APIClient) SendFriendRequest(username string) error {
	data := map[string]string{"username": username}
	resp, err := c.post("/api/friends/request", data, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		if msg, ok := result["error"].(string); ok {
			return fmt.Errorf(msg)
		}
		return fmt.Errorf("failed to send request")
	}
	return nil
}

func (c *APIClient) GetFriendRequests() ([]models.FriendRequest, error) {
	resp, err := c.get("/api/friends/requests")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var requests []models.FriendRequest
	json.NewDecoder(resp.Body).Decode(&requests)
	if requests == nil {
		requests = []models.FriendRequest{}
	}
	return requests, nil
}

func (c *APIClient) AcceptFriendRequest(requestID string) error {
	data := map[string]string{"request_id": requestID}
	resp, err := c.post("/api/friends/accept", data, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to accept request")
	}
	return nil
}

func (c *APIClient) RemoveFriend(friendID string) error {
	data := map[string]string{"friend_id": friendID}
	body, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", c.BaseURL+"/api/friends/remove", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		if msg, ok := result["error"].(string); ok {
			return fmt.Errorf(msg)
		}
		return fmt.Errorf("failed to remove friend")
	}
	return nil
}

func (c *APIClient) SearchUsers(query string) ([]models.User, error) {
	path := fmt.Sprintf("/api/users/search?q=%s", url.QueryEscape(query))
	resp, err := c.get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var users []models.User
	json.NewDecoder(resp.Body).Decode(&users)
	if users == nil {
		users = []models.User{}
	}
	return users, nil
}

func (c *APIClient) get(path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+path, nil)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.http.Do(req)
}

func (c *APIClient) post(path string, data interface{}, auth bool) (*http.Response, error) {
	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", c.BaseURL+path, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if auth && c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return c.http.Do(req)
}
