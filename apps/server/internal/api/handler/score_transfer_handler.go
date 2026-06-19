package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/dto"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
)

type ScoreTransferHandler struct {
	scoreTransfer *scoretransfersvc.Service
	query         *querysvc.Service
}

func NewScoreTransferHandler(scoreTransfer *scoretransfersvc.Service, query *querysvc.Service) *ScoreTransferHandler {
	return &ScoreTransferHandler{scoreTransfer: scoreTransfer, query: query}
}

func (h *ScoreTransferHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req dto.SubmitScoreTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, err)
		return
	}
	result, err := h.scoreTransfer.Submit(r.Context(), middleware.UserID(r.Context()), chi.URLParam(r, "id"), scoretransfersvc.SubmitInput{
		ReceiverPlayerIDs: req.ReceiverPlayerIDs,
		Amount:            req.Amount,
		IdempotencyKey:    req.IdempotencyKey,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *ScoreTransferHandler) List(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var beforeSequenceNo *int
	if v := r.URL.Query().Get("beforeSequenceNo"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			beforeSequenceNo = &n
		}
	}
	result, err := h.query.ListScoreTransfers(r.Context(), middleware.UserID(r.Context()), gameID, beforeSequenceNo, limit)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
