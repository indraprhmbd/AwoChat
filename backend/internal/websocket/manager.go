package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/gorilla/websocket"
)

const (
	MessageTypeMessage = "message"
	MessageTypeTyping  = "typing"
	MessageTypeError   = "error"
	MessageTypeAuth    = "auth"
)

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

type Connection struct {
	UserID     string
	RoomID     string
	Conn       *websocket.Conn
	Send       chan []byte
	Ctx        context.Context
	Cancel     context.CancelFunc
	lastTyping time.Time
}

type Room struct {
	ID          string
	mu          sync.RWMutex
	connections map[string]*Connection
	typingUsers map[string]time.Time
}

type Manager struct {
	mu       sync.RWMutex
	rooms    map[string]*Room
	limits   config.LimitsConfig
	stopChan chan struct{}
}

func NewManager(limits config.LimitsConfig) *Manager {
	m := &Manager{
		rooms:    make(map[string]*Room),
		limits:   limits,
		stopChan: make(chan struct{}),
	}

	go m.cleanupTypingIndicators()

	return m
}

func (m *Manager) Stop() {
	close(m.stopChan)
}

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

func (m *Manager) GetRoom(roomID string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[roomID]
}

func (m *Manager) AddConnection(conn *Connection) {
	room := m.GetOrCreateRoom(conn.RoomID)

	room.mu.Lock()
	defer room.mu.Unlock()

	room.connections[conn.UserID] = conn
}

func (m *Manager) RemoveConnection(conn *Connection) {
	room := m.GetRoom(conn.RoomID)
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	delete(room.connections, conn.UserID)
	delete(room.typingUsers, conn.UserID)

	if len(room.connections) == 0 {
		m.mu.Lock()
		delete(m.rooms, room.ID)
		m.mu.Unlock()
	}
}

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

		select {
		case conn.Send <- message:
		default:
			go conn.Cancel()
		}
	}
}

func (m *Manager) SetTypingUser(roomID, userID string) {
	room := m.GetRoom(roomID)
	if room == nil {
		return
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	room.typingUsers[userID] = time.Now()
}

func (m *Manager) GetTypingUsers(roomID string) []string {
	room := m.GetRoom(roomID)
	if room == nil {
		return nil
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	expiry := time.Now().Add(-6 * time.Second)
	var typing []string
	for userID, lastTyping := range room.typingUsers {
		if lastTyping.After(expiry) {
			typing = append(typing, userID)
		}
	}

	return typing
}

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

func (m *Manager) UpdateLastTyping(conn *Connection) {
	conn.lastTyping = time.Now()
}

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

var ErrBufferFull = Error("send buffer full")

type Error string

func (e Error) Error() string {
	return string(e)
}
