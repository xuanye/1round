package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	playersvc "github.com/xuanye/one-round/apps/server/internal/app/player"
)

type PlayerHandler struct {
	player *playersvc.Service
}

func NewPlayerHandler(player *playersvc.Service) *PlayerHandler {
	return &PlayerHandler{player: player}
}

func (h *PlayerHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req dto.PlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.player.Add(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), req.DisplayName)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *PlayerHandler) List(w http.ResponseWriter, r *http.Request) {
	result, err := h.player.List(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *PlayerHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req dto.PlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.player.Update(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), chi.URLParam(r, "playerId"), req.DisplayName)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *PlayerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	err := h.player.Delete(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), chi.URLParam(r, "playerId"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.DeletePlayerResponse{Deleted: true})
}
