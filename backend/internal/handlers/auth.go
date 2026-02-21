package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
	"github.com/indraprhmbd/AwoChat/backend/internal/models"
	"github.com/indraprhmbd/AwoChat/backend/internal/ratelimiter"
	"github.com/indraprhmbd/AwoChat/backend/internal/websocket"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db           *database.DB
	cfg          *config.Config
	wsManager    *websocket.Manager
	signupLimiter *ratelimiter.RateLimiter
	loginLimiter  *ratelimiter.RateLimiter
}

func NewAuthHandler(db *database.DB, cfg *config.Config, wsManager *websocket.Manager) *AuthHandler {
	return &AuthHandler{
		db:            db,
		cfg:           cfg,
		wsManager:     wsManager,
		signupLimiter: ratelimiter.New(cfg.Limits.SignupRateLimit, time.Minute),
		loginLimiter:  ratelimiter.New(cfg.Limits.LoginRateLimit, time.Minute),
	}
}

// SignupRequest represents the signup request body
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Signup handles user registration
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting by IP
	ip := getClientIP(r)
	if !h.signupLimiter.Allow(ip) {
		http.Error(w, "Too many signup attempts, please try again later", http.StatusTooManyRequests)
		return
	}

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validation
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create user
	ctx := r.Context()
	user, err := h.db.CreateUser(ctx, req.Email, string(hashedPassword))
	if err != nil {
		if err == database.ErrUserExists {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create session
	expiresAt := time.Now().Add(h.cfg.Session.Expiration)
	session, err := h.db.CreateSession(ctx, user.ID, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.Session.CookieName,
		Value:    session.ID,
		Expires:  expiresAt,
		HttpOnly: h.cfg.Session.CookieHTTPOnly,
		Secure:   h.cfg.Session.CookieSecure,
		SameSite: parseSameSite(h.cfg.Session.CookieSameSite),
		Path:     "/",
	})

	// Return user info with session token (for WebSocket auth)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           user.ID,
		"email":        user.Email,
		"created_at":   user.CreatedAt,
		"session_token": session.ID,
	})
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Rate limiting by IP
	ip := getClientIP(r)
	if !h.loginLimiter.Allow(ip) {
		http.Error(w, "Too many login attempts, please try again later", http.StatusTooManyRequests)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user by email
	ctx := r.Context()
	user, err := h.db.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == database.ErrUserNotFound {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Delete any existing sessions for this user
	h.db.DeleteUserSessions(ctx, user.ID)

	// Create new session
	expiresAt := time.Now().Add(h.cfg.Session.Expiration)
	session, err := h.db.CreateSession(ctx, user.ID, expiresAt)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.Session.CookieName,
		Value:    session.ID,
		Expires:  expiresAt,
		HttpOnly: h.cfg.Session.CookieHTTPOnly,
		Secure:   h.cfg.Session.CookieSecure,
		SameSite: parseSameSite(h.cfg.Session.CookieSameSite),
		Path:     "/",
	})

	// Return user info with session token (for WebSocket auth)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            user.ID,
		"email":         user.Email,
		"created_at":    user.CreatedAt,
		"session_token": session.ID,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get session from cookie
	sessionID, err := r.Cookie(h.cfg.Session.CookieName)
	if err == nil && sessionID.Value != "" {
		// Delete session from database
		h.db.DeleteSession(r.Context(), sessionID.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.Session.CookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.cfg.Session.CookieSecure,
		SameSite: parseSameSite(h.cfg.Session.CookieSameSite),
		Path:     "/",
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out"))
}

// Me returns the current authenticated user
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
}

// getSessionUser retrieves the user from the session cookie
func getSessionUser(r *http.Request, db *database.DB, cfg *config.Config) *models.User {
	cookie, err := r.Cookie(cfg.Session.CookieName)
	if err != nil {
		return nil
	}

	session, err := db.GetSession(r.Context(), cookie.Value)
	if err != nil {
		return nil
	}

	user, err := db.GetUserByID(r.Context(), session.UserID)
	if err != nil {
		return nil
	}

	return user
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for reverse proxy setups)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// parseSameSite converts string to http.SameSite
func parseSameSite(s string) http.SameSite {
	switch s {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	case "Lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteDefaultMode
	}
}
