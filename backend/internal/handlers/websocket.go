package handlers

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
	"github.com/indraprhmbd/AwoChat/backend/internal/ratelimiter"
	ws "github.com/indraprhmbd/AwoChat/backend/internal/websocket"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowedOrigins := []string{
			"https://awochat.indraprhmbd.my.id",
			"http://awochat.indraprhmbd.my.id",
		}
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
	HandshakeTimeout: 45 * time.Second,
}

type WebSocketHandler struct {
	wsManager      *ws.Manager
	db             *database.DB
	cfg            *config.Config
	messageLimiter *ratelimiter.RateLimiter
}

func NewWebSocketHandler(wsManager *ws.Manager, db *database.DB, cfg *config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager:      wsManager,
		db:             db,
		cfg:            cfg,
		messageLimiter: ratelimiter.New(cfg.Limits.MessageRateLimit, time.Second),
	}
}

func (h *WebSocketHandler) Upgrade(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie(h.cfg.Session.CookieName)

	if err != nil || sessionCookie.Value == "" {
		sessionCookie = &http.Cookie{
			Name:  h.cfg.Session.CookieName,
			Value: r.URL.Query().Get("token"),
		}
	}

	user, room, role, err := GetRoomAndMemberRole(r, h.db, h.cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	wsConn := &ws.Connection{
		UserID: user.ID,
		RoomID: room.ID,
		Conn:   conn,
		Send:   make(chan []byte, h.cfg.Limits.MaxSendBufferSize),
		Ctx:    ctx,
		Cancel: cancel,
	}

	h.wsManager.AddConnection(wsConn)

	go h.readLoop(wsConn, room.ID, user.ID, role)
	go h.writeLoop(wsConn)
}

func (h *WebSocketHandler) readLoop(conn *ws.Connection, roomID, userID, role string) {
	defer func() {
		conn.Cancel()
		h.wsManager.RemoveConnection(conn)
		conn.Conn.Close()
	}()

	conn.Conn.SetReadLimit(int64(h.cfg.Limits.MaxMessageSize))

	for {
		select {
		case <-conn.Ctx.Done():
			return
		default:
			_, message, err := conn.Conn.ReadMessage()
			if err != nil {
				return
			}

			if err := h.handleMessage(conn, roomID, userID, role, message); err != nil {
				h.sendError(conn, err.Error())
			}
		}
	}
}

func (h *WebSocketHandler) writeLoop(conn *ws.Connection) {
	defer func() {
		conn.Conn.Close()
	}()

	for {
		select {
		case <-conn.Ctx.Done():
			return
		case message, ok := <-conn.Send:
			if !ok {
				return
			}

			conn.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (h *WebSocketHandler) handleMessage(conn *ws.Connection, roomID, userID, role string, data []byte) error {
	var msg ws.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return Error("invalid message format")
	}

	switch msg.Type {
	case ws.MessageTypeMessage:
		return h.handleChatMessage(conn, roomID, userID, msg.Content)
	case ws.MessageTypeTyping:
		return h.handleTyping(conn, roomID, userID)
	default:
		return Error("unknown message type")
	}
}

func (h *WebSocketHandler) handleChatMessage(conn *ws.Connection, roomID, userID, content string) error {
	ip := userID
	if !h.messageLimiter.Allow(ip) {
		return Error("rate limit exceeded")
	}

	if content == "" || len(content) > h.cfg.Limits.MaxMessageSize {
		return Error("invalid message content")
	}

	content = html.EscapeString(content)

	ctx, cancel := context.WithTimeout(conn.Ctx, 5*time.Second)
	defer cancel()

	message, err := h.db.CreateMessage(ctx, roomID, userID, content)
	if err != nil {
		return Error("failed to save message")
	}

	user, err := h.db.GetUserByID(ctx, userID)
	if err != nil {
		return Error("failed to get user")
	}

	response := ws.WSMessage{
		Type:      ws.MessageTypeMessage,
		ID:        message.ID,
		UserID:    message.UserID,
		UserEmail: user.Email,
		Content:   message.Content,
		CreatedAt: message.CreatedAt,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return Error("failed to marshal message")
	}

	h.wsManager.BroadcastToRoom(roomID, data, userID)

	return ws.SendMessage(conn, &response)
}

func (h *WebSocketHandler) handleTyping(conn *ws.Connection, roomID, userID string) error {
	if !h.wsManager.CanSendTyping(roomID, userID, h.cfg.Limits.TypingThrottleSec) {
		return nil
	}

	h.wsManager.UpdateLastTyping(conn)
	h.wsManager.SetTypingUser(roomID, userID)

	response := ws.WSMessage{
		Type:   ws.MessageTypeTyping,
		UserID: userID,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return Error("failed to marshal typing indicator")
	}

	h.wsManager.BroadcastToRoom(roomID, data, userID)

	return nil
}

func (h *WebSocketHandler) sendError(conn *ws.Connection, message string) {
	response := ws.WSMessage{
		Type:  ws.MessageTypeError,
		Error: message,
	}

	data, _ := json.Marshal(response)
	select {
	case conn.Send <- data:
	default:
	}
}
