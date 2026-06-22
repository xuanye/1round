package middleware

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

// statusWriter wraps http.ResponseWriter to capture status code and error.
type statusWriter struct {
	http.ResponseWriter
	status    int
	err       error
	errorCode int
	userID    string
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) SetError(err error, code int) {
	w.err = err
	w.errorCode = code
}

func (w *statusWriter) SetUserID(userID string) {
	w.userID = userID
}

// Hijack implements http.Hijacker so that websocket.Accept can take over
// the underlying TCP connection. Without this, the coder/websocket library
// returns 501 Not Implemented.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func RequestLog(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			duration := time.Since(start)

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", sw.status),
				zap.Duration("duration", duration),
			}

			if sw.userID != "" {
				fields = append(fields, zap.String("user_id", sw.userID))
			}
			if gameSessionID := gameSessionIDFromRequest(r); gameSessionID != "" {
				fields = append(fields, zap.String("game_session_id", gameSessionID))
			}
			if sw.err != nil {
				fields = append(fields, zap.Error(sw.err), zap.Int("error_code", sw.errorCode))
			}

			switch logLevelForRequest(sw.status, sw.err) {
			case "error":
				log.Error("request", fields...)
			case "warn":
				log.Warn("request", fields...)
			default:
				log.Info("request", fields...)
			}
		})
	}
}

func logLevelForRequest(status int, err error) string {
	switch {
	case status >= 500:
		return "error"
	case err == nil:
		return "info"
	case isExpectedBusinessError(err):
		return "info"
	default:
		return "warn"
	}
}

// Keep this list in sync with internal/domain/errors.go when adding new
// sentinel business errors that should log as expected client rejections.
func isExpectedBusinessError(err error) bool {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument),
		errors.Is(err, domain.ErrInvalidPlayer),
		errors.Is(err, domain.ErrGameSessionFinished),
		errors.Is(err, domain.ErrActiveGameExists),
		errors.Is(err, domain.ErrGameCapacityFull),
		errors.Is(err, domain.ErrDuplicateDisplayName),
		errors.Is(err, domain.ErrCannotLeaveWithNonZeroScore),
		errors.Is(err, domain.ErrParticipantInactive),
		errors.Is(err, domain.ErrIdempotencyConflict),
		errors.Is(err, domain.ErrFinishRequestPending),
		errors.Is(err, domain.ErrFinishRequestNotPending),
		errors.Is(err, domain.ErrScoreTotalMustBeZero),
		errors.Is(err, domain.ErrInvalidScoreTransferAmount),
		errors.Is(err, domain.ErrScoreTransferReceiverRequired),
		errors.Is(err, domain.ErrIdempotencyKeyRequired),
		errors.Is(err, domain.ErrAlreadyDeactivated):
		return true
	default:
		return false
	}
}

func gameSessionIDFromRequest(r *http.Request) string {
	path := r.URL.Path
	if strings.Contains(path, "/game-sessions/") || strings.Contains(path, "/ws/game-sessions/") {
		return chi.URLParam(r, "id")
	}
	return ""
}
