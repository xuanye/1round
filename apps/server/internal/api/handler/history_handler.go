package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
)

type HistoryHandler struct {
	query *querysvc.Service
}

func NewHistoryHandler(query *querysvc.Service) *HistoryHandler {
	return &HistoryHandler{query: query}
}

func (h *HistoryHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	var beforeSettledAt *time.Time
	if raw := r.URL.Query().Get("beforeSettledAt"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(w, err)
			return
		}
		beforeSettledAt = &t
	}

	result, err := h.query.History(r.Context(), userID, beforeSettledAt, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *HistoryHandler) Detail(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	gameSessionID := chi.URLParam(r, "id")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	var beforeSequenceNo *int
	if raw := r.URL.Query().Get("beforeSequenceNo"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(w, err)
			return
		}
		beforeSequenceNo = &n
	}

	result, err := h.query.SettlementDetail(r.Context(), userID, gameSessionID, beforeSequenceNo, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *HistoryHandler) PublicSettlement(w http.ResponseWriter, r *http.Request) {
	shareToken := chi.URLParam(r, "shareToken")

	result, err := h.query.PublicSettlement(r.Context(), shareToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *HistoryHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	result, err := h.query.UserStats(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}


func parseLimit(raw string, defaultLimit int) int {
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > 100 {
		n = 100
	}
	return n
}
