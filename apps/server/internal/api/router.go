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
	roundsvc "github.com/xuanye/one-round/apps/server/internal/app/round"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
)

type Services struct {
	Auth      *authsvc.Service
	Game      *gamesvc.Service
	Player    *playersvc.Service
	Round     *roundsvc.Service
	Query     *querysvc.Service
	Tokens    *jwtauth.JWTService
	WebSocket *handler.WebSocketHandler
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
	gameHandler := handler.NewGameHandler(services.Game, services.Query)
	playerHandler := handler.NewPlayerHandler(services.Player)
	roundHandler := handler.NewRoundHandler(services.Round, services.Query)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/wechat-login", authHandler.WechatLogin)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(services.Tokens))
			r.Post("/game-sessions", gameHandler.Create)
			r.Get("/game-sessions/current", gameHandler.Current)
			r.Post("/game-sessions/join", gameHandler.Join)
			r.Post("/game-sessions/join-preview", gameHandler.JoinPreview)
			r.Get("/game-sessions/{id}", gameHandler.Get)
			r.Get("/game-sessions/{id}/summary", gameHandler.Summary)
			r.Post("/game-sessions/{id}/finish", gameHandler.Finish)
			r.Patch("/game-sessions/{id}/my-profile", playerHandler.UpdateMyProfile)
			r.Post("/game-sessions/{id}/players", playerHandler.Add)
			r.Get("/game-sessions/{id}/players", playerHandler.List)
			r.Patch("/game-sessions/{id}/players/{playerId}", playerHandler.Update)
			r.Delete("/game-sessions/{id}/players/{playerId}", playerHandler.Delete)
			r.Post("/game-sessions/{id}/rounds", roundHandler.Submit)
			r.Get("/game-sessions/{id}/rounds/recent", roundHandler.Recent)
			r.Get("/game-sessions/{id}/ranking", gameHandler.Ranking)
		})
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(services.Tokens))
		r.Get("/ws/game-sessions/{id}", services.WebSocket.Connect)
	})
	return r
}
