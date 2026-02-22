package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
	"github.com/indraprhmbd/AwoChat/backend/internal/models"
	"github.com/indraprhmbd/AwoChat/backend/internal/ratelimiter"
	"github.com/indraprhmbd/AwoChat/backend/internal/websocket"
)

type RoomHandler struct {
	db          *database.DB
	cfg         *config.Config
	wsManager   *websocket.Manager
	joinLimiter *ratelimiter.RateLimiter
}

func NewRoomHandler(db *database.DB, cfg *config.Config, wsManager *websocket.Manager) *RoomHandler {
	return &RoomHandler{
		db:          db,
		cfg:         cfg,
		wsManager:   wsManager,
		joinLimiter: ratelimiter.New(cfg.Limits.LoginRateLimit, time.Minute),
	}
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

func (h *RoomHandler) HandleRooms(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Room name is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	room, err := h.db.CreateRoom(ctx, req.Name, user.ID)
	if err != nil {
		log.Printf("Error creating room: %v (user: %s, name: %s)", err, user.ID, req.Name)
		http.Error(w, "Failed to create room: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           room.ID,
		"name":         room.Name,
		"owner_id":     room.OwnerID,
		"invite_token": room.InviteToken,
		"created_at":   room.CreatedAt,
	})
}

func (h *RoomHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	rooms, err := h.db.GetUserRooms(ctx, user.ID)
	if err != nil {
		http.Error(w, "Failed to get rooms", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(rooms))
	for i, room := range rooms {
		response[i] = map[string]interface{}{
			"id":       room.ID,
			"name":     room.Name,
			"owner_id": room.OwnerID,
			"created_at": room.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := r.URL.Path[len("/api/rooms/"):]
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	isMember, err := h.db.IsRoomMember(ctx, user.ID, roomID)
	if err != nil {
		log.Printf("Error checking membership: %v", err)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	if !isMember {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	room, err := h.db.GetRoomByID(ctx, roomID)
	if err != nil {
		if err == database.ErrRoomNotFound {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting room: %v", err)
		http.Error(w, "Failed to get room: "+err.Error(), http.StatusInternalServerError)
		return
	}

	members, err := h.db.GetRoomMembers(ctx, roomID)
	if err != nil {
		log.Printf("Error getting members: %v", err)
		http.Error(w, "Failed to get members: "+err.Error(), http.StatusInternalServerError)
		return
	}

	memberList := make([]map[string]interface{}, len(members))
	for i, member := range members {
		role, _ := h.db.GetRoomMemberRole(ctx, member.ID, roomID)
		memberList[i] = map[string]interface{}{
			"id":    member.ID,
			"email": member.Email,
			"role":  role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       room.ID,
		"name":     room.Name,
		"owner_id": room.OwnerID,
		"members":  memberList,
	})
}

type JoinRequest struct {
	Token string `json:"token"`
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := getClientIP(r)
	if !h.joinLimiter.Allow(ip) {
		http.Error(w, "Too many join attempts, please try again later", http.StatusTooManyRequests)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Invite token is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	room, err := h.db.GetRoomByInviteToken(ctx, req.Token)
	if err != nil {
		if err == database.ErrInviteTokenInvalid {
			http.Error(w, "Invalid invite token", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to find room", http.StatusInternalServerError)
		return
	}

	err = h.db.AddRoomMember(ctx, user.ID, room.ID, "member", h.cfg.Limits.MaxRoomMembers)
	if err != nil {
		log.Printf("Error adding member: %v", err)
		if err == database.ErrAlreadyMember {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":   room.ID,
				"name": room.Name,
			})
			return
		}
		if err == database.ErrRoomFull {
			http.Error(w, "Room is full", http.StatusConflict)
			return
		}
		log.Printf("Error joining room: %v", err)
		http.Error(w, "Failed to join room: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   room.ID,
		"name": room.Name,
	})
}

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		RoomID string `json:"room_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	isMember, _ := h.db.IsRoomMember(ctx, user.ID, req.RoomID)
	if !isMember {
		http.Error(w, "Not a member of this room", http.StatusForbidden)
		return
	}

	room, err := h.db.GetRoomByID(ctx, req.RoomID)
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	if room.OwnerID == user.ID {
		http.Error(w, "Room owner must transfer ownership or delete room before leaving", http.StatusBadRequest)
		return
	}

	if err := h.db.RemoveRoomMember(ctx, user.ID, req.RoomID); err != nil {
		http.Error(w, "Failed to leave room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Left room successfully"))
}

func (h *RoomHandler) Members(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	isMember, _ := h.db.IsRoomMember(ctx, user.ID, roomID)
	if !isMember {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	members, err := h.db.GetRoomMembers(ctx, roomID)
	if err != nil {
		http.Error(w, "Failed to get members", http.StatusInternalServerError)
		return
	}

	memberList := make([]map[string]interface{}, len(members))
	for i, member := range members {
		role, _ := h.db.GetRoomMemberRole(ctx, member.ID, roomID)
		memberList[i] = map[string]interface{}{
			"id":    member.ID,
			"email": member.Email,
			"role":  role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memberList)
}

type UpdateRoomRequest struct {
	Name string `json:"name"`
}

func (h *RoomHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	isAdmin, err := h.db.IsRoomAdmin(ctx, user.ID, roomID)
	if err != nil || !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	var req UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Room name is required", http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateRoomName(ctx, roomID, req.Name); err != nil {
		log.Printf("Error updating room: %v", err)
		http.Error(w, "Failed to update room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Room updated"))
}

func (h *RoomHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	isAdmin, err := h.db.IsRoomAdmin(ctx, user.ID, roomID)
	if err != nil || !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	if err := h.db.DeleteRoom(ctx, roomID); err != nil {
		log.Printf("Error deleting room: %v", err)
		http.Error(w, "Failed to delete room", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Room deleted"))
}

func (h *RoomHandler) InviteLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSessionUser(r, h.db, h.cfg)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	room, err := h.db.GetRoomByID(ctx, roomID)
	if err != nil {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}

	if room.OwnerID != user.ID {
		http.Error(w, "Owner access required", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"invite_token": room.InviteToken,
	})
}

func GetRoomAndMemberRole(r *http.Request, db *database.DB, cfg *config.Config) (*models.User, *models.Room, string, error) {
	user := getSessionUser(r, db, cfg)
	if user == nil {
		return nil, nil, "", ErrUnauthorized
	}

	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		return nil, nil, "", ErrRoomIDRequired
	}

	ctx := r.Context()

	room, err := db.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, nil, "", ErrRoomNotFound
	}

	role, err := db.GetRoomMemberRole(ctx, user.ID, roomID)
	if err != nil {
		return nil, nil, "", ErrNotMember
	}

	return user, room, role, nil
}

var (
	ErrUnauthorized       = Error("unauthorized")
	ErrRoomIDRequired     = Error("room_id is required")
	ErrRoomNotFound       = Error("room not found")
	ErrNotMember          = Error("not a member of this room")
)
