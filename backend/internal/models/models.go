package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Room struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerID     string    `json:"owner_id"`
	InviteToken string    `json:"-"` // Never expose in API
	CreatedAt   time.Time `json:"created_at"`
}

type RoomMember struct {
	UserID  string    `json:"user_id"`
	RoomID  string    `json:"room_id"`
	Role    string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"-"`
	UserID    string    `json:"-"`
	ExpiresAt time.Time `json:"-"`
	CreatedAt time.Time `json:"-"`
}
