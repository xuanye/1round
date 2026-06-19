package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/xuanye/one-round/apps/server/internal/api"
	wshandler "github.com/xuanye/one-round/apps/server/internal/api/handler"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
)

func TestFakeAuthCreateJoinAddSubmitSummaryRankingAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, Round: app.round, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	token := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, token, "/api/game-sessions", map[string]any{"name": "家庭聚会", "zeroSumRequired": true})
	gameID := game["id"].(string)
	inviteCode := game["inviteCode"].(string)
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/join", map[string]any{"inviteCode": inviteCode})
	p1 := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "A"})
	p2 := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "B"})
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/rounds", map[string]any{
		"scores": []map[string]any{{"playerId": p1["id"], "score": 5}, {"playerId": p2["id"], "score": -5}},
	})
	summary := getJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/summary")
	if summary["roundCount"].(float64) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	ranking := getJSON[[]any](t, router, token, "/api/game-sessions/"+gameID+"/ranking")
	if len(ranking) != 2 {
		t.Fatalf("unexpected ranking: %+v", ranking)
	}
}

func loginHTTP(t *testing.T, router http.Handler, code string) string {
	t.Helper()
	result := postJSON[map[string]any](t, router, "", "/api/auth/wechat-login", map[string]any{"code": code})
	return result["token"].(string)
}

func postJSON[T any](t *testing.T, router http.Handler, token, path string, payload any) T {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("POST %s status %d body %s", path, rec.Code, rec.Body.String())
	}
	return decodeData[T](t, rec.Body.Bytes())
}

func getJSON[T any](t *testing.T, router http.Handler, token, path string) T {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("GET %s status %d body %s", path, rec.Code, rec.Body.String())
	}
	return decodeData[T](t, rec.Body.Bytes())
}

func decodeData[T any](t *testing.T, raw []byte) T {
	t.Helper()
	var envelope struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 {
		t.Fatalf("unexpected code %d body %s", envelope.Code, string(raw))
	}
	var data T
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data
}
