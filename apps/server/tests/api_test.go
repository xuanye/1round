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
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	token := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, token, "/api/game-sessions", map[string]any{"name": "家庭聚会"})
	gameID := game["id"].(string)
	inviteCode := game["inviteCode"].(string)
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/join", map[string]any{"inviteCode": inviteCode})
	p1 := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "A"})
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/players", map[string]any{"displayName": "B"})
	// Owner gives 5 to player A via score transfer
	_ = postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{p1["id"].(string)},
		"amount":            5,
		"idempotencyKey":    "api-test-1",
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
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
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
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
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

	// Get joiner's player ID
	joinerPlayer := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join-preview", map[string]any{"inviteCode": game["inviteCode"]})
	_ = joinerPlayer

	// List players to find joiner's player ID
	players := getJSON[[]map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/players")
	var joinerPlayerID string
	for _, p := range players {
		if p["userId"] != nil && p["displayName"] == "妈妈" {
			joinerPlayerID = p["id"].(string)
		}
	}

	// Owner gives 5 to joiner via score transfer (makes owner's score non-zero)
	postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{joinerPlayerID},
		"amount":            5,
		"idempotencyKey":    "leave-test-api",
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
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
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

func TestPublicSettlementAPI(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player,
		ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	token := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, token, "/api/game-sessions", map[string]any{"name": "结算测试"})
	gameID := game["id"].(string)

	// Finish the game to generate a public share token
	finished := postJSON[map[string]any](t, router, token, "/api/game-sessions/"+gameID+"/finish", map[string]any{})
	shareToken := finished["publicShareToken"].(string)

	// Verify handler extracts shareToken from URL path and returns proper response shape
	req := httptest.NewRequest(http.MethodGet, "/api/public/settlements/"+shareToken, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			GameSessionID string `json:"gameSessionId"`
			Name          string `json:"name"`
			Participants  []struct {
				DisplayName string `json:"displayName"`
				FinalScore  int    `json:"finalScore"`
			} `json:"participants"`
			ScoreTransfers []struct {
				ID     string `json:"id"`
				Amount int    `json:"amount"`
			} `json:"scoreTransfers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if envelope.Code != 0 {
		t.Fatalf("unexpected code %d", envelope.Code)
	}
	if envelope.Data.GameSessionID != gameID {
		t.Fatalf("expected gameSessionID %s, got %s", gameID, envelope.Data.GameSessionID)
	}
	if len(envelope.Data.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(envelope.Data.Participants))
	}
	if len(envelope.Data.ScoreTransfers) != 0 {
		t.Fatalf("expected 0 score transfers, got %d", len(envelope.Data.ScoreTransfers))
	}

	// Verify 404 for invalid share token
	req2 := httptest.NewRequest(http.MethodGet, "/api/public/settlements/invalid-token", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid token, got %d body %s", rec2.Code, rec2.Body.String())
	}
}
