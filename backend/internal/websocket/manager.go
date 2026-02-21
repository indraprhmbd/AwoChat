package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/gorilla/websocket"
)

// Message types for WebSocket communication
const (
	MessageTypeMessage = "message"
	MessageTypeTyping  = "typing"
	MessageTypeError   = "error"
	MessageTypeAuth    = "auth"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string      `json:"type"`
	Content   string      `json:"content,omitempty"`
	ID        int64       `json:"id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	UserEmail string      `json:"user_email,omitempty"`
	RoomID    string      `json:"room_id,omitempty"`
	CreatedAt time.Time   `json:"created_at,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// Connection represents a single WebSocket connection
type Connection struct {
	UserID    string
	RoomID    string
	Conn      *websocket.Conn
	Send      chan []byte
	Ctx       context.Context
	Cancel    context.CancelFunc
	lastTyping time.Time
}

// Room represents a chat room with active connections
type Room struct {
	ID          string
	mu          sync.RWMutex
	connections map[string]*Connection // userID -> Connection
	typingUsers map[string]time.Time   // userID -> lastTypingTime
}

// Manager manages all rooms and connections
type Manager struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	limits   config.LimitsConfig
	stopChan chan struct{}
}

// NewManager creates a new WebSocket manager
func NewManager(limits config.LimitsConfig) *Manager {
	m := &Manager{
		rooms:    make(map[string]*Room),
		limits:   limits,
		stopChan: make(chan struct{}),
	}

	// Start typing indicator cleanup goroutine
	go m.cleanupTypingIndicators()

	return m
}

// Stop stops the manager and all cleanup goroutines
func (m *Manager) Stop() {
	close(m.stopChan)
}

// GetOrCreateRoom gets an existing room or creates a new one
func (m *Manager) GetOrCreateRoom(roomID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.rooms[roomID]
	if !exists {
		room = &Room{
			ID:          roomID,
			connections: make(map[string]*Connection),
			typingUsers: make(map[string]time.Time),
		}
		m.rooms[roomID] = room
	}

	return room
}

// GetRoom gets a room by ID (returns nil if not exists)
func (m *Manager) GetRoom(roomID string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[roomID]
}

// AddConnection adds a connection to a room
func (m *Manager) AddConnection(conn *Connection) {
	room := m.GetOrCreateRoom(conn.RoomID)

	room.mu.Lock()
	defer room.mu.Unlock()

	room.connections[conn.UserID] = conn
}

// RemoveConnection removes a connection from its room
func (m *Manager) RemoveConnection(conn *Connection) {
	room := m.GetRoom(conn.RoomID)
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	delete(room.connections, conn.UserID)
	delete(room.typingUsers, conn.UserID)

	// Clean up empty rooms
	if len(room.connections) == 0 {
		m.mu.Lock()
		delete(m.rooms, room.ID)
		m.mu.Unlock()
	}
}

// BroadcastToRoom broadcasts a message to all connections in a room
func (m *Manager) BroadcastToRoom(roomID string, message []byte, excludeUserID string) {
	room := m.GetRoom(roomID)
	if room == nil {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for userID, conn := range room.connections {
		if userID == excludeUserID {
			continue
		}

		// Non-blocking send with backpressure handling
		select {
		case conn.Send <- message:
		default:
			// Client is slow, disconnect them
			go conn.Cancel()
		}
	}
}

// SetTypingUser marks a user as typing in a room
func (m *Manager) SetTypingUser(roomID, userID string) {
	room := m.GetRoom(roomID)
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	room.typingUsers[userID] = time.Now()
}

// GetTypingUsers returns all users currently typing in a room
func (m *Manager) GetTypingUsers(roomID string) []string {
	room := m.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	// Filter expired typing indicators (older than 6 seconds)
	expiry := time.Now().Add(-6 * time.Second)
	var typing []string
	for userID, lastTyping := range room.typingUsers {
		if lastTyping.After(expiry) {
			typing = append(typing, userID)
		}
	}

	return typing
}

// CanSendTyping checks if enough time has passed since last typing event
func (m *Manager) CanSendTyping(roomID, userID string, throttleSec int) bool {
	room := m.GetRoom(roomID)
	if room == nil {
		return true
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	conn, exists := room.connections[userID]
	if !exists {
		return true
	}

	return time.Since(conn.lastTyping) >= time.Duration(throttleSec)*time.Second
}

// UpdateLastTyping updates the last typing time for a connection
func (m *Manager) UpdateLastTyping(conn *Connection) {
	conn.lastTyping = time.Now()
}

// cleanupTypingIndicators periodically removes expired typing indicators
func (m *Manager) cleanupTypingIndicators() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.mu.RLock()
			for _, room := range m.rooms {
				room.mu.Lock()
				expiry := time.Now().Add(-6 * time.Second)
				for userID, lastTyping := range room.typingUsers {
					if lastTyping.Before(expiry) {
						delete(room.typingUsers, userID)
					}
				}
				room.mu.Unlock()
			}
			m.mu.RUnlock()
		}
	}
}

// Stats returns current manager statistics
func (m *Manager) Stats() struct {
	ActiveConnections int
	ActiveRooms       int
	Goroutines        int
} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connections := 0
	for _, room := range m.rooms {
		room.mu.RLock()
		connections += len(room.connections)
		room.mu.RUnlock()
	}

	return struct {
		ActiveConnections int
		ActiveRooms       int
		Goroutines        int
	}{
		ActiveConnections: connections,
		ActiveRooms:       len(m.rooms),
		Goroutines:        0,
	}
}

// SendMessage serializes and sends a WSMessage
func SendMessage(conn *Connection, msg *WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case conn.Send <- data:
		return nil
	default:
		return ErrBufferFull
	}
}

// ErrBufferFull indicates the send buffer is full
var ErrBufferFull = Error("send buffer full")

// Error is a custom error type
type Error string

func (e Error) Error() string {
	return string(e)
}
