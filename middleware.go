package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/kataras/iris/v12"
)

// CSRF Cookie name
const CSRFCookieName = "_ct"

// setCSRFCookie sets the CSRF token in a dedicated cookie
func setCSRFCookie(ctx iris.Context, token string) {
	ctx.SetCookieKV(CSRFCookieName, token,
		iris.CookiePath("/"),
		iris.CookieHTTPOnly(false),
	)
}

// getCSRFCookie reads the CSRF token from the dedicated cookie
func getCSRFCookie(ctx iris.Context) string {
	return ctx.GetCookie(CSRFCookieName)
}

// CSRFMiddleware validates CSRF tokens on POST/PUT/DELETE requests.
// Token generation is handled by the global middleware (SetCSRFToken).
// This middleware only does validation.
func CSRFMiddleware() iris.Handler {
	return func(ctx iris.Context) {
		method := ctx.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" || method == "TRACE" {
			ctx.Next()
			return
		}

		// Get stored token from dedicated cookie (not session)
		storedToken := getCSRFCookie(ctx)

		// Get token from form or header
		submittedToken := ctx.FormValue("csrf_token")
		if submittedToken == "" {
			submittedToken = ctx.GetHeader("X-CSRF-Token")
		}

		log.Printf("[DEBUG] CSRF validate: stored=%q, submitted=%q", storedToken, submittedToken)

		// Validate token
		if storedToken == "" || submittedToken == "" || storedToken != submittedToken {
			log.Printf("[WARN] CSRF token mismatch: stored=%q, submitted=%q", storedToken, submittedToken)
			ctx.StatusCode(http.StatusForbidden)
			ctx.ViewData("Error", ErrMsgCSRFInvalid)
			ctx.View("error.html")
			return
		}

		ctx.Next()
	}
}

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("[ERROR] Failed to generate CSRF token: %v", err)
		return ""
	}
	return hex.EncodeToString(bytes)
}

// SetCSRFToken sets a CSRF token in the dedicated cookie (only generates new if none exists)
// This is called once per request in the global middleware.
func SetCSRFToken(ctx iris.Context) string {
	token := getCSRFCookie(ctx)
	if token == "" {
		token = GenerateCSRFToken()
	}
	setCSRFCookie(ctx, token)
	ctx.Values().Set("csrfToken", token)
	return token
}

// FlashMessage represents a flash message
type FlashMessage struct {
	Type    string // "error" or "success"
	Message string
}

// SetFlash sets a flash message
func SetFlash(ctx iris.Context, msgType, message string) {
	session := Sess.Start(ctx)
	session.SetFlash("flashType", msgType)
	session.SetFlash("flashMessage", message)
}

// GetFlash gets and clears a flash message
func GetFlash(ctx iris.Context) FlashMessage {
	session := Sess.Start(ctx)
	msgType := session.GetFlashStringDefault("flashType", "")
	message := session.GetFlashStringDefault("flashMessage", "")
	return FlashMessage{Type: msgType, Message: message}
}

// TemplateHelpers returns helper functions for templates
func TemplateHelpers() map[string]interface{} {
	return map[string]interface{}{
		"csrfField": func(token string) string {
			return `<input type="hidden" name="csrf_token" value="` + token + `">`
		},
	}
}
