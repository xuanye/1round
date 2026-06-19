package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *T     `json:"data"`
}

func JSON[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse[T]{Code: 0, Message: "ok", Data: &data})
}

func Empty(w http.ResponseWriter) {
	JSON(w, http.StatusOK, map[string]any{})
}

func Error(w http.ResponseWriter, err error) {
	code, status, msg := mapError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse[any]{Code: code, Message: msg, Data: nil})
}

func mapError(err error) (int, int, string) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument), errors.Is(err, domain.ErrInvalidPlayer), errors.Is(err, domain.ErrScoreTotalMustBeZero), errors.Is(err, domain.ErrCannotLeaveWithNonZeroScore), errors.Is(err, domain.ErrOwnerRequired), errors.Is(err, domain.ErrInvalidScoreTransferAmount), errors.Is(err, domain.ErrScoreTransferReceiverRequired), errors.Is(err, domain.ErrIdempotencyKeyRequired), errors.Is(err, domain.ErrParticipantInactive):
		return 40001, http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return 40101, http.StatusUnauthorized, err.Error()
	case errors.Is(err, domain.ErrForbidden), errors.Is(err, domain.ErrGameMemberRequired), errors.Is(err, domain.ErrParticipantRequired):
		return 40301, http.StatusForbidden, err.Error()
	case errors.Is(err, domain.ErrNotFound):
		return 40401, http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrGameSessionFinished), errors.Is(err, domain.ErrAlreadyDeactivated), errors.Is(err, domain.ErrIdempotencyConflict):
		return 40901, http.StatusConflict, err.Error()
	default:
		return 50001, http.StatusInternalServerError, "internal error"
	}
}
