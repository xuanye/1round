package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
)

type GameHandler struct {
	game  *gamesvc.Service
	query *querysvc.Service
}

func NewGameHandler(game *gamesvc.Service, query *querysvc.Service) *GameHandler {
	return &GameHandler{game: game, query: query}
}

func (h *GameHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.game.Create(r.Context(), middleware.UserID(r.Context()), req.Name, req.ZeroSumRequired)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *GameHandler) Join(w http.ResponseWriter, r *http.Request) {
	var req dto.JoinGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	id, err := h.game.Join(r.Context(), middleware.UserID(r.Context()), req.InviteCode)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"gameSessionId": id})
}

func (h *GameHandler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.game.GetForMember(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *GameHandler) Summary(w http.ResponseWriter, r *http.Request) {
	result, err := h.query.Summary(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *GameHandler) Ranking(w http.ResponseWriter, r *http.Request) {
	result, err := h.query.Ranking(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *GameHandler) Finish(w http.ResponseWriter, r *http.Request) {
	result, err := h.game.Finish(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"id": result.ID, "status": string(result.Status)})
}
