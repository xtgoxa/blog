package main

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID        int64
	Username  string
	Email     string
	Password  string
	Role      string
	CreatedAt time.Time
}

// Comment represents a comment on a post
type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Content   string
	Status    string
	CreatedAt time.Time
	Username  string
}

// Custom errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserExists        = errors.New("user already exists")
	ErrInvalidCredential = errors.New("invalid credentials")
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword checks if the password matches the hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser creates a new user
func CreateUser(username, email, password string) (*User, error) {
	// Check if user exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ? OR email = ?", username, email).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	result, err := DB.Exec(
		"INSERT INTO users (username, email, password) VALUES (?, ?, ?)",
		username, email, hashedPassword,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &User{
		ID:       id,
		Username: username,
		Email:    email,
		Role:     RoleUser,
	}, nil
}

// GetUserByUsername gets a user by username
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := DB.QueryRow(
		"SELECT id, username, email, password, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// GetUserByID gets a user by ID
func GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := DB.QueryRow(
		"SELECT id, username, email, password, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

// CreateComment creates a new comment
func CreateComment(postID, userID int64, content string) (*Comment, error) {
	result, err := DB.Exec(
		"INSERT INTO comments (post_id, user_id, content, status) VALUES (?, ?, ?, ?)",
		postID, userID, content, StatusPending,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Comment{
		ID:      id,
		PostID:  postID,
		UserID:  userID,
		Content: content,
		Status:  StatusPending,
	}, nil
}

// GetCommentsByPostID gets approved comments for a post (public view)
func GetCommentsByPostID(postID int64) ([]Comment, error) {
	rows, err := DB.Query(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.status, c.created_at, u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ? AND c.status = ?
		ORDER BY c.created_at DESC
	`, postID, StatusApproved)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.Status, &c.CreatedAt, &c.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// GetAllComments gets all comments (admin view)
func GetAllComments() ([]Comment, error) {
	rows, err := DB.Query(`
		SELECT c.id, c.post_id, c.user_id, c.content, c.status, c.created_at, u.username
		FROM comments c
		JOIN users u ON c.user_id = u.id
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.Status, &c.CreatedAt, &c.Username)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// ApproveComment approves a comment
func ApproveComment(id int64) error {
	_, err := DB.Exec("UPDATE comments SET status = ? WHERE id = ?", StatusApproved, id)
	return err
}

// RejectComment rejects a comment
func RejectComment(id int64) error {
	_, err := DB.Exec("UPDATE comments SET status = ? WHERE id = ?", StatusRejected, id)
	return err
}

// DeleteComment deletes a comment
func DeleteComment(id int64) error {
	_, err := DB.Exec("DELETE FROM comments WHERE id = ?", id)
	return err
}

// CountPendingComments counts pending comments
func CountPendingComments() (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM comments WHERE status = ?", StatusPending).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAllUsers gets all users
func GetAllUsers() ([]User, error) {
	rows, err := DB.Query(`
		SELECT id, username, email, password, role, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateUserRole updates a user's role
func UpdateUserRole(id int64, role string) error {
	_, err := DB.Exec("UPDATE users SET role = ? WHERE id = ?", role, id)
	return err
}

// DeleteUser deletes a user
func DeleteUser(id int64) error {
	// First delete user's comments
	_, _ = DB.Exec("DELETE FROM comments WHERE user_id = ?", id)
	// Then delete user
	_, err := DB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// CountUsers counts total users
func CountUsers() (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountUsersByRole counts users by role
func CountUsersByRole(role string) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = ?", role).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAdminPassword gets the hashed admin password from settings
func GetAdminPassword() (string, error) {
	var password string
	err := DB.QueryRow("SELECT key_value FROM settings WHERE key_name = 'admin_password'").Scan(&password)
	if err != nil {
		return "", err
	}
	return password, nil
}

// UpdateAdminPassword updates the admin password
func UpdateAdminPassword(hashedPassword string) error {
	_, err := DB.Exec("UPDATE settings SET key_value = ? WHERE key_name = 'admin_password'", hashedPassword)
	return err
}
