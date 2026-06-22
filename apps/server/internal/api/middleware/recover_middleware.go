package middleware

import (
	"net/http"

	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

func Recover(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Error("request panic", zap.Any("value", recovered))
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
