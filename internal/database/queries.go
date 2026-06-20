package database

import (
	"fmt"
	"terminal-chat/internal/models"
)

// CreateUser inserts a new user and returns it
func (d *DB) CreateUser(username, passwordHash string) (*models.User, error) {
	user := &models.User{}
	err := d.Conn.QueryRow(
		`INSERT INTO users (username, password_hash) 
		 VALUES ($1, $2) 
		 RETURNING id, username, created_at`,
		username, passwordHash,
	).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByUsername returns a user and their password hash by username
func (d *DB) GetUserByUsername(username string) (*models.User, string, error) {
	user := &models.User{}
	var passwordHash string
	err := d.Conn.QueryRow(
		`SELECT id, username, password_hash, is_online, last_seen 
		 FROM users 
		 WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &passwordHash, &user.IsOnline, &user.LastSeen)
	if err != nil {
		return nil, "", err
	}
	return user, passwordHash, nil
}

// GetUserByID returns a user by ID
func (d *DB) GetUserByID(userID string) (*models.User, error) {
	user := &models.User{}
	err := d.Conn.QueryRow(
		`SELECT id, username, is_online, last_seen 
		 FROM users 
		 WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Username, &user.IsOnline, &user.LastSeen)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// SetUserOnline updates the online status of a user
func (d *DB) SetUserOnline(userID string, online bool) error {
	_, err := d.Conn.Exec(
		`UPDATE users SET is_online = $1, last_seen = NOW() WHERE id = $2`,
		online, userID,
	)
	return err
}

// SaveMessage saves a new message and returns it
func (d *DB) SaveMessage(fromUserID, toUserID, content string) (*models.Message, error) {
	msg := &models.Message{}
	err := d.Conn.QueryRow(
		`INSERT INTO messages (from_user_id, to_user_id, content) 
		 VALUES ($1, $2, $3) 
		 RETURNING id, from_user_id, to_user_id, content, created_at`,
		fromUserID, toUserID, content,
	).Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetConversation returns messages between two users
func (d *DB) GetConversation(userID, friendID string, limit int) ([]models.Message, error) {
	rows, err := d.Conn.Query(
		`SELECT id, from_user_id, to_user_id, content, created_at 
		 FROM messages 
		 WHERE (from_user_id = $1 AND to_user_id = $2) 
		    OR (from_user_id = $2 AND to_user_id = $1)
		 ORDER BY created_at ASC 
		 LIMIT $3`,
		userID, friendID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &msg.Content, &msg.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// SendFriendRequest creates a new friend request
func (d *DB) SendFriendRequest(fromUserID, toUserID string) (string, error) {
	// Check if they are already friends
	var count int
	d.Conn.QueryRow(
		`SELECT COUNT(*) FROM friends 
		 WHERE (user_id = $1 AND friend_id = $2) 
		    OR (user_id = $2 AND friend_id = $1)`,
		fromUserID, toUserID,
	).Scan(&count)
	if count > 0 {
		return "", fmt.Errorf("already friends")
	}

	// Check if there's a pending request from the other direction (auto-accept)
	var existingID string
	err := d.Conn.QueryRow(
		`SELECT id FROM friend_requests 
		 WHERE from_user_id = $1 AND to_user_id = $2 AND status = 'pending'`,
		toUserID, fromUserID,
	).Scan(&existingID)

	if err == nil {
		// Auto-accept the existing request
		return existingID, d.AcceptFriendRequest(existingID, toUserID, fromUserID)
	}

	// Delete any old requests between them (rejected or from this direction)
	d.Conn.Exec(
		`DELETE FROM friend_requests 
		 WHERE (from_user_id = $1 AND to_user_id = $2) 
		    OR (from_user_id = $2 AND to_user_id = $1 AND status != 'pending')`,
		fromUserID, toUserID,
	)

	// Create new request
	var requestID string
	err = d.Conn.QueryRow(
		`INSERT INTO friend_requests (from_user_id, to_user_id) 
		 VALUES ($1, $2) 
		 RETURNING id`,
		fromUserID, toUserID,
	).Scan(&requestID)

	return requestID, err
}

// GetPendingRequests returns pending friend requests for a user
func (d *DB) GetPendingRequests(userID string) ([]models.FriendRequest, error) {
	rows, err := d.Conn.Query(
		`SELECT fr.id, fr.from_user_id, u.username, fr.status, fr.created_at
		 FROM friend_requests fr
		 INNER JOIN users u ON fr.from_user_id = u.id
		 WHERE fr.to_user_id = $1 AND fr.status = 'pending'
		 ORDER BY fr.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.FriendRequest
	for rows.Next() {
		var req models.FriendRequest
		if err := rows.Scan(&req.ID, &req.FromUserID, &req.FromUsername, &req.Status, &req.CreatedAt); err != nil {
			continue
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// AcceptFriendRequest accepts a friend request and creates friendship
func (d *DB) AcceptFriendRequest(requestID, fromUserID, toUserID string) error {
	tx, err := d.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update request status
	_, err = tx.Exec(
		`UPDATE friend_requests SET status = 'accepted' WHERE id = $1`,
		requestID,
	)
	if err != nil {
		return err
	}

	// Create bidirectional friendship
	_, err = tx.Exec(
		`INSERT INTO friends (user_id, friend_id) VALUES ($1, $2), ($2, $1) ON CONFLICT DO NOTHING`,
		fromUserID, toUserID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveFriend removes a friendship and all messages between users
func (d *DB) RemoveFriend(userID, friendID string) error {
	tx, err := d.Conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove friendship
	_, err = tx.Exec(
		`DELETE FROM friends WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		userID, friendID,
	)
	if err != nil {
		return err
	}

	// Delete messages between them
	_, err = tx.Exec(
		`DELETE FROM messages WHERE (from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1)`,
		userID, friendID,
	)
	if err != nil {
		return err
	}

	// Reject any pending requests between them
	_, err = tx.Exec(
		`UPDATE friend_requests SET status = 'rejected' 
		 WHERE (from_user_id = $1 AND to_user_id = $2) OR (from_user_id = $2 AND to_user_id = $1)`,
		userID, friendID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetFriends returns all friends of a user
func (d *DB) GetFriends(userID string) ([]models.User, error) {
	rows, err := d.Conn.Query(
		`SELECT u.id, u.username, u.is_online, u.last_seen
		 FROM users u
		 INNER JOIN friends f ON u.id = f.friend_id
		 WHERE f.user_id = $1
		 ORDER BY u.is_online DESC, u.username ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []models.User
	for rows.Next() {
		var friend models.User
		if err := rows.Scan(&friend.ID, &friend.Username, &friend.IsOnline, &friend.LastSeen); err != nil {
			continue
		}
		friends = append(friends, friend)
	}
	return friends, nil
}

// SearchUsers searches users by username
func (d *DB) SearchUsers(query, excludeUserID string) ([]models.User, error) {
	rows, err := d.Conn.Query(
		`SELECT id, username, is_online 
		 FROM users 
		 WHERE username ILIKE $1 AND id != $2 
		 LIMIT 20`,
		"%"+query+"%", excludeUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.IsOnline); err != nil {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}