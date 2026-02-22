package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/indraprhmbd/AwoChat/backend/internal/database"
	"github.com/indraprhmbd/AwoChat/backend/internal/handlers"
	"github.com/indraprhmbd/AwoChat/backend/internal/middleware"
	"github.com/indraprhmbd/AwoChat/backend/internal/websocket"
)

func main() {
	cfg := config.Load()

	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	wsManager := websocket.NewManager(cfg.Limits)

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go database.StartSessionCleanup(cleanupCtx, db, cfg.Session.Expiration)

	authHandler := handlers.NewAuthHandler(db, cfg, wsManager)
	roomHandler := handlers.NewRoomHandler(db, cfg, wsManager)
	messageHandler := handlers.NewMessageHandler(db, cfg, wsManager)
	wsHandler := handlers.NewWebSocketHandler(wsManager, db, cfg)

	mux := http.NewServeMux()
	handler := middleware.SecurityHeaders(mux)

	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics(wsManager))

	mux.HandleFunc("/api/auth/signup", authHandler.Signup)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/me", authHandler.Me)

	mux.HandleFunc("/api/rooms", roomHandler.HandleRooms)
	mux.HandleFunc("/api/rooms/", roomHandler.Get)
	mux.HandleFunc("/api/rooms/join", roomHandler.Join)
	mux.HandleFunc("/api/rooms/leave", roomHandler.Leave)
	mux.HandleFunc("/api/rooms/members", roomHandler.Members)
	mux.HandleFunc("/api/rooms/update", roomHandler.Update)
	mux.HandleFunc("/api/rooms/delete", roomHandler.Delete)
	mux.HandleFunc("/api/rooms/invite", roomHandler.InviteLink)

	mux.HandleFunc("/api/messages", messageHandler.List)

	mux.HandleFunc("/ws", wsHandler.Upgrade)

	mux.Handle("/", http.FileServer(http.Dir("../frontend/dist")))

	server := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-shutdown
	log.Println("Shutting down server...")

	cleanupCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func handleMetrics(wsManager *websocket.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := wsManager.Stats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `{"active_connections":%d,"active_rooms":%d,"goroutines":%d}`,
			stats.ActiveConnections, stats.ActiveRooms, runtime.NumGoroutine())
	}
}
