package database

import (
	"context"
	"errors"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomExists        = errors.New("room already exists")
	ErrNotRoomMember     = errors.New("user is not a member of this room")
	ErrNotRoomAdmin      = errors.New("user is not an admin of this room")
	ErrRoomFull          = errors.New("room has reached maximum members")
	ErrAlreadyMember     = errors.New("user is already a member of this room")
	ErrInviteTokenInvalid = errors.New("invalid invite token")
)

// CreateRoom creates a new room with the given owner
func (db *DB) CreateRoom(ctx context.Context, name, ownerID string) (*models.Room, error) {
	room := &models.Room{
		ID:          uuid.New().String(),
		Name:        name,
		OwnerID:     ownerID,
		InviteToken: generateInviteToken(),
		CreatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Insert room
	err = tx.QueryRow(
		ctx,
		"INSERT INTO rooms (id, name, owner_id, invite_token) VALUES ($1, $2, $3, $4) RETURNING created_at",
		room.ID, room.Name, room.OwnerID, room.InviteToken,
	).Scan(&room.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Add owner as admin member
	_, err = tx.Exec(
		ctx,
		"INSERT INTO room_members (user_id, room_id, role) VALUES ($1, $2, 'admin')",
		ownerID, room.ID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return room, nil
}

// GetRoomByID retrieves a room by ID
func (db *DB) GetRoomByID(ctx context.Context, id string) (*models.Room, error) {
	room := &models.Room{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, name, owner_id, invite_token, created_at FROM rooms WHERE id = $1",
		id,
	).Scan(&room.ID, &room.Name, &room.OwnerID, &room.InviteToken, &room.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}

	return room, nil
}

// GetRoomByInviteToken retrieves a room by invite token
func (db *DB) GetRoomByInviteToken(ctx context.Context, token string) (*models.Room, error) {
	room := &models.Room{}

	err := db.Pool.QueryRow(
		ctx,
		"SELECT id, name, owner_id, invite_token, created_at FROM rooms WHERE invite_token = $1",
		token,
	).Scan(&room.ID, &room.Name, &room.OwnerID, &room.InviteToken, &room.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInviteTokenInvalid
		}
		return nil, err
	}

	return room, nil
}

// GetUserRooms retrieves all rooms a user is a member of
func (db *DB) GetUserRooms(ctx context.Context, userID string) ([]*models.Room, error) {
	rows, err := db.Pool.Query(
		ctx,
		`SELECT r.id, r.name, r.owner_id, r.invite_token, r.created_at 
		 FROM rooms r 
		 JOIN room_members rm ON r.id = rm.room_id 
		 WHERE rm.user_id = $1 
		 ORDER BY r.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		room := &models.Room{}
		if err := rows.Scan(&room.ID, &room.Name, &room.OwnerID, &room.InviteToken, &room.CreatedAt); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, rows.Err()
}

// GetRoomMemberCount returns the number of members in a room
func (db *DB) GetRoomMemberCount(ctx context.Context, roomID string) (int, error) {
	var count int
	err := db.Pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1",
		roomID,
	).Scan(&count)
	return count, err
}

// IsRoomMember checks if a user is a member of a room
func (db *DB) IsRoomMember(ctx context.Context, userID, roomID string) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM room_members WHERE user_id = $1 AND room_id = $2)",
		userID, roomID,
	).Scan(&exists)
	return exists, err
}

// GetRoomMemberRole returns the role of a user in a room
func (db *DB) GetRoomMemberRole(ctx context.Context, userID, roomID string) (string, error) {
	var role string
	err := db.Pool.QueryRow(
		ctx,
		"SELECT role FROM room_members WHERE user_id = $1 AND room_id = $2",
		userID, roomID,
	).Scan(&role)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotRoomMember
		}
		return "", err
	}

	return role, nil
}

// AddRoomMember adds a user to a room with the given role
func (db *DB) AddRoomMember(ctx context.Context, userID, roomID, role string, maxMembers int) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Lock the room row to prevent concurrent modifications
	_, err = tx.Exec(ctx, "SELECT 1 FROM rooms WHERE id = $1 FOR UPDATE", roomID)
	if err != nil {
		return err
	}

	// Check current member count (no lock needed since room is locked)
	var count int
	err = tx.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM room_members WHERE room_id = $1",
		roomID,
	).Scan(&count)

	if err != nil {
		return err
	}

	if count >= maxMembers {
		return ErrRoomFull
	}

	// Check if already a member
	var exists bool
	err = tx.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM room_members WHERE user_id = $1 AND room_id = $2)",
		userID, roomID,
	).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		return ErrAlreadyMember
	}

	// Add member
	_, err = tx.Exec(
		ctx,
		"INSERT INTO room_members (user_id, room_id, role) VALUES ($1, $2, $3)",
		userID, roomID, role,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RemoveRoomMember removes a user from a room
func (db *DB) RemoveRoomMember(ctx context.Context, userID, roomID string) error {
	_, err := db.Pool.Exec(
		ctx,
		"DELETE FROM room_members WHERE user_id = $1 AND room_id = $2",
		userID, roomID,
	)
	return err
}

// GetRoomMembers retrieves all members of a room with their details
func (db *DB) GetRoomMembers(ctx context.Context, roomID string) ([]*models.User, error) {
	rows, err := db.Pool.Query(
		ctx,
		`SELECT u.id, u.email, u.created_at, rm.role, rm.joined_at 
		 FROM users u 
		 JOIN room_members rm ON u.id = rm.user_id 
		 WHERE rm.room_id = $1 
		 ORDER BY rm.joined_at ASC`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type UserWithRole struct {
		User     *models.User
		Role     string
		JoinedAt time.Time
	}

	var members []*models.User
	for rows.Next() {
		user := &models.User{}
		var role string
		var joinedAt time.Time
		if err := rows.Scan(&user.ID, &user.Email, &user.CreatedAt, &role, &joinedAt); err != nil {
			return nil, err
		}
		_ = role    // Can be used if needed
		_ = joinedAt // Can be used if needed
		members = append(members, user)
	}

	return members, rows.Err()
}

// IsRoomAdmin checks if a user is an admin of a room
func (db *DB) IsRoomAdmin(ctx context.Context, userID, roomID string) (bool, error) {
	var isAdmin bool
	err := db.Pool.QueryRow(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM room_members WHERE user_id = $1 AND room_id = $2 AND role = 'admin')",
		userID, roomID,
	).Scan(&isAdmin)
	return isAdmin, err
}

// UpdateRoomName updates the name of a room
func (db *DB) UpdateRoomName(ctx context.Context, roomID, name string) error {
	_, err := db.Pool.Exec(ctx, "UPDATE rooms SET name = $1 WHERE id = $2", name, roomID)
	return err
}

// DeleteRoom deletes a room and all its messages/members
func (db *DB) DeleteRoom(ctx context.Context, roomID string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM rooms WHERE id = $1", roomID)
	return err
}

// generateInviteToken generates a random invite token (36 chars)
func generateInviteToken() string {
	return uuid.New().String()
}
