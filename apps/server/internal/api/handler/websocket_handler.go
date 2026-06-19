package handler

import (
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	"github.com/xuanye/one-round/apps/server/internal/realtime"
)

type WebSocketHandler struct {
	game         *gamesvc.Service
	hub          realtime.Hub
	queueSize    int
	writeTimeout time.Duration
}

func NewWebSocketHandler(game *gamesvc.Service, hub realtime.Hub, queueSize int, writeTimeout time.Duration) *WebSocketHandler {
	return &WebSocketHandler{game: game, hub: hub, queueSize: queueSize, writeTimeout: writeTimeout}
}

func (h *WebSocketHandler) Connect(w http.ResponseWriter, r *http.Request) {
	gameSessionID := chi.URLParam(r, "id")
	userID := middleware.UserID(r.Context())
	if err := h.game.RequireMember(r.Context(), userID, gameSessionID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	client := &realtime.Client{ID: uuid.NewString(), UserID: userID, GameSessionID: gameSessionID, Conn: conn, Send: make(chan realtime.Event, h.queueSize)}
	if err := h.hub.Register(r.Context(), gameSessionID, client); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "register failed")
		return
	}
	go client.WriteLoop(r.Context(), h.writeTimeout, func() { h.hub.Unregister(gameSessionID, client) })
	client.ReadLoop(r.Context(), func() { h.hub.Unregister(gameSessionID, client) })
}
