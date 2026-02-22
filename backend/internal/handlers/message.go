package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
)

type MessageHandler struct {
	db  *database.DB
	cfg *config.Config
}

func NewMessageHandler(db *database.DB, cfg *config.Config, wsManager interface{}) *MessageHandler {
	return &MessageHandler{
		db:  db,
		cfg: cfg,
	}
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
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

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := r.Context()

	isMember, _ := h.db.IsRoomMember(ctx, user.ID, roomID)
	if !isMember {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	messages, err := h.db.GetMessagesWithUserDetails(ctx, roomID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
