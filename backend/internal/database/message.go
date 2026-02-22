package database

import (
	"context"
	"errors"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/models"
	"github.com/jackc/pgx/v5"
)

var (
	ErrMessageNotFound = errors.New("message not found")
)

func (db *DB) CreateMessage(ctx context.Context, roomID, userID, content string) (*models.Message, error) {
	msg := &models.Message{
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	err := db.Pool.QueryRow(
		ctx,
		"INSERT INTO messages (room_id, user_id, content) VALUES ($1, $2, $3) RETURNING id, created_at",
		msg.RoomID, msg.UserID, msg.Content,
	).Scan(&msg.ID, &msg.CreatedAt)

	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (db *DB) GetMessagesByRoom(ctx context.Context, roomID string, limit int, offset int) ([]*models.Message, error) {
	rows, err := db.Pool.Query(
		ctx,
		`SELECT id, room_id, user_id, content, created_at
		 FROM messages
		 WHERE room_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		roomID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*models.Message{}
	for rows.Next() {
		msg := &models.Message{}
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.UserID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}

func (db *DB) GetMessageByID(ctx context.Context, id int64) (*models.Message, error) {
	msg := &models.Message{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, room_id, user_id, content, created_at FROM messages WHERE id = $1",
		id,
	).Scan(&msg.ID, &msg.RoomID, &msg.UserID, &msg.Content, &msg.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return msg, nil
}

func (db *DB) GetMessagesWithUserDetails(ctx context.Context, roomID string, limit int, offset int) ([]map[string]interface{}, error) {
	rows, err := db.Pool.Query(
		ctx,
		`SELECT m.id, m.room_id, m.user_id, m.content, m.created_at, u.email as user_email
		 FROM messages m
		 LEFT JOIN users u ON m.user_id = u.id
		 WHERE m.room_id = $1
		 ORDER BY m.created_at DESC
		 LIMIT $2 OFFSET $3`,
		roomID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var msg models.Message
		var userEmail string
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.UserID, &msg.Content, &msg.CreatedAt, &userEmail); err != nil {
			return nil, err
		}
		messages = append(messages, map[string]interface{}{
			"id":         msg.ID,
			"room_id":    msg.RoomID,
			"user_id":    msg.UserID,
			"user_email": userEmail,
			"content":    msg.Content,
			"created_at": msg.CreatedAt,
		})
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, rows.Err()
}
