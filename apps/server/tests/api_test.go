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
		Auth: app.auth, Game: app.game, Player: app.player, Round: app.round, ScoreTransfer: app.scoreTransfer, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	token := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, token, "/api/game-sessions", map[string]any{"name": "家庭聚会"})
	gameID := game["id"].(string)
	inviteCode := game["inviteCode"].(string)
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/join", map[string]any{"inviteCode": inviteCode})
	// List existing players (owner player is auto-created)
	existingPlayers := getJSON[[]map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players")
	p1 := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "A"})
	p2 := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "B"})
	scores := []map[string]any{{"playerId": p1["id"], "score": 5}, {"playerId": p2["id"], "score": -5}}
	for _, ep := range existingPlayers {
		scores = append(scores, map[string]any{"playerId": ep["id"], "score": 0})
	}
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/rounds", map[string]any{
		"scores": scores,
	})
	summary := getJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/summary")
	if summary["scoreTransferCount"].(float64) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	ranking := getJSON[[]any](t, router, token, "/api/game-sessions/"+gameID+"/ranking")
	if len(ranking) != 3 {
		t.Fatalf("unexpected ranking: %+v", ranking)
	}
}

func TestLeaveGameAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, Round: app.round, ScoreTransfer: app.scoreTransfer, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	ownerToken := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{"name": "家庭聚会"})
	gameID := game["id"].(string)

	// Leave with zero score should succeed
	result := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/leave", map[string]any{})
	if result["left"] != true {
		t.Fatalf("expected left=true, got %+v", result)
	}
}

func TestLeaveRequiresZeroScoreAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, Round: app.round, ScoreTransfer: app.scoreTransfer, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	ownerToken := loginHTTP(t, router, "owner-code")
	joinerToken := loginHTTP(t, router, "joiner-code")
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{"name": "家庭聚会"})
	gameID := game["id"].(string)

	// Join with joiner
	joinerGameID := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{
		"inviteCode": game["inviteCode"], "displayName": "妈妈",
	})
	if joinerGameID["gameSessionId"] != gameID {
		t.Fatalf("unexpected join result: %+v", joinerGameID)
	}

	// Submit a round to give owner non-zero score
	existingPlayers := getJSON[[]map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/players")
	scores := []map[string]any{}
	for _, p := range existingPlayers {
		scores = append(scores, map[string]any{"playerId": p["id"], "score": 0})
	}
	// Replace owner score with non-zero
	for i, p := range existingPlayers {
		if p["userId"] != nil {
			scores[i] = map[string]any{"playerId": p["id"], "score": 5}
		}
	}
	postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/rounds", map[string]any{
		"scores": scores,
	})

	// Try to leave with non-zero score - should fail with 400
	req, _ := http.NewRequest(http.MethodPost, "/api/game-sessions/"+gameID+"/leave", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 status, got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestCurrentPreviewJoinAndProfileAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	ownerToken := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{"name": "家庭聚会", "maxParticipants": 2})
	current := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/current")
	if current["id"] != game["id"] {
		t.Fatalf("unexpected current game: %+v", current)
	}

	joinerToken := loginHTTP(t, router, "joiner-code")
	preview := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join-preview", map[string]any{"inviteCode": game["inviteCode"]})
	if preview["participantCount"].(float64) != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	joined := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{"inviteCode": game["inviteCode"], "displayName": "妈妈"})
	if joined["gameSessionId"] != game["id"] {
		t.Fatalf("unexpected join: %+v", joined)
	}
	updated := patchJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/"+game["id"].(string)+"/my-profile", map[string]any{"displayName": "阿姨"})
	if updated["displayName"] != "阿姨" {
		t.Fatalf("unexpected profile: %+v", updated)
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

func patchJSON[T any](t *testing.T, router http.Handler, token, path string, payload any) T {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("PATCH %s status %d body %s", path, rec.Code, rec.Body.String())
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
