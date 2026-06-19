package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/redhu/one-round/apps/server/internal/api/dto"
	"github.com/redhu/one-round/apps/server/internal/api/middleware"
	"github.com/redhu/one-round/apps/server/internal/api/response"
	querysvc "github.com/redhu/one-round/apps/server/internal/app/query"
	roundsvc "github.com/redhu/one-round/apps/server/internal/app/round"
)

type RoundHandler struct {
	round *roundsvc.Service
	query *querysvc.Service
}

func NewRoundHandler(round *roundsvc.Service, query *querysvc.Service) *RoundHandler {
	return &RoundHandler{round: round, query: query}
}

func (h *RoundHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req dto.SubmitRoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.round.Submit(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), req.Scores, req.Note)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *RoundHandler) Recent(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.query.RecentRounds(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
