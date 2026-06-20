package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/handler"
	"github.com/xuanye/one-round/apps/server/internal/api/middleware"
	authsvc "github.com/xuanye/one-round/apps/server/internal/app/auth"
	gamesvc "github.com/xuanye/one-round/apps/server/internal/app/game"
	playersvc "github.com/xuanye/one-round/apps/server/internal/app/player"
	querysvc "github.com/xuanye/one-round/apps/server/internal/app/query"
	scoretransfersvc "github.com/xuanye/one-round/apps/server/internal/app/scoretransfer"
	settlementsvc "github.com/xuanye/one-round/apps/server/internal/app/settlement"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
)

type Services struct {
	Auth          *authsvc.Service
	Game          *gamesvc.Service
	Player        *playersvc.Service
	ScoreTransfer *scoretransfersvc.Service
	Settlement    *settlementsvc.Service
	Query         *querysvc.Service
	Tokens        *jwtauth.JWTService
	WebSocket     *handler.WebSocketHandler
}

func NewRouter(logger *slog.Logger, services Services) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Use(middleware.RequestLog(logger))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	authHandler := handler.NewAuthHandler(services.Auth)
	gameHandler := handler.NewGameHandler(services.Game, services.Query, services.Player, services.Settlement)
	playerHandler := handler.NewPlayerHandler(services.Player)
	scoreTransferHandler := handler.NewScoreTransferHandler(services.ScoreTransfer, services.Query)
	historyHandler := handler.NewHistoryHandler(services.Query)

	r.Get("/api/public/settlements/{shareToken}", historyHandler.PublicSettlement)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/wechat-login", authHandler.WechatLogin)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(services.Tokens))
			r.Post("/game-sessions", gameHandler.Create)
			r.Get("/game-sessions/current", gameHandler.Current)
			r.Post("/game-sessions/join", gameHandler.Join)
			r.Post("/game-sessions/join-preview", gameHandler.JoinPreview)
			r.Get("/game-sessions/{id}/join-mini-program-code", gameHandler.JoinMiniProgramCode)
			r.Get("/game-sessions/{id}", gameHandler.Get)
			r.Get("/game-sessions/{id}/summary", gameHandler.Summary)
			r.Post("/game-sessions/{id}/finish", gameHandler.Finish)
			r.Post("/game-sessions/{id}/finish-requests", gameHandler.RequestFinish)
			r.Post("/game-sessions/{id}/finish-requests/{requestId}/approve", gameHandler.ApproveFinishRequest)
			r.Post("/game-sessions/{id}/finish-requests/{requestId}/reject", gameHandler.RejectFinishRequest)
			r.Post("/game-sessions/{id}/leave", gameHandler.Leave)
			r.Patch("/game-sessions/{id}/my-profile", playerHandler.UpdateMyProfile)
			r.Post("/game-sessions/{id}/score-transfers", scoreTransferHandler.Submit)
			r.Get("/game-sessions/{id}/score-transfers", scoreTransferHandler.List)
			r.Get("/ranking", gameHandler.Ranking)
			r.Get("/history/game-sessions", historyHandler.List)
			r.Get("/history/game-sessions/{id}", historyHandler.Detail)
			r.Get("/history/stats", historyHandler.Stats)
		})
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(services.Tokens))
		r.Get("/ws/game-sessions/{id}", services.WebSocket.Connect)
	})
	return r
}
