package main

// Constants for the application
const (
	// Comment status
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"

	// User roles
	RoleAdmin = "admin"
	RoleUser  = "user"

	// Session keys
	SessionUserID    = "userID"
	SessionUsername  = "username"
	SessionIsAdmin   = "isAdmin"
	SessionCSRFToken = "csrfToken"

	// Flash message keys
	FlashError   = "flashError"
	FlashSuccess = "flashSuccess"
)

// Error messages (English to avoid encoding issues)
var (
	ErrMsgInvalidCredentials = "Invalid username or password"
	ErrMsgPasswordMismatch   = "Passwords do not match"
	ErrMsgUserExists        = "Username or email already exists"
	ErrMsgSystemError       = "System error, please try again later"
	ErrMsgUnauthorized      = "Please login first"
	ErrMsgForbidden        = "Access denied"
	ErrMsgNotFound         = "Resource not found"
	ErrMsgInvalidInput     = "Invalid input"
	ErrMsgCSRFInvalid      = "Invalid request, please refresh and try again"
)

// Success messages
var (
	MsgLoginSuccess    = "Login successful"
	MsgRegisterSuccess = "Registration successful"
	MsgLogoutSuccess   = "Logged out successfully"
	MsgPostCreated     = "Post created successfully"
	MsgPostUpdated     = "Post updated successfully"
	MsgPostDeleted     = "Post deleted successfully"
	MsgCommentPending  = "Comment submitted, awaiting approval"
	MsgPasswordChanged = "Password changed successfully"
)
