package middleware

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

// statusWriter wraps http.ResponseWriter to capture status code and error.
type statusWriter struct {
	http.ResponseWriter
	status int
	err    error
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) SetError(err error) {
	w.err = err
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

			if sw.err != nil {
				fields = append(fields, zap.Error(sw.err))
			}

			switch {
			case sw.status >= 500:
				log.Error("request", fields...)
			case sw.status >= 400:
				log.Warn("request", fields...)
			default:
				log.Info("request", fields...)
			}
		})
	}
}
