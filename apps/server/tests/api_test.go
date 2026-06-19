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

	ownerToken := loginHTTP(t, router, "owner-code")
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{"name": "家庭聚会"})
	gameID := game["id"].(string)
	inviteCode := game["inviteCode"].(string)

	// Joiner joins the game
	joinerToken := loginHTTP(t, router, "joiner-code")
	_ = postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{"inviteCode": inviteCode, "displayName": "妈妈"})

	// Get joiner's player ID from summary
	summary := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/summary")
	players := summary["players"].([]any)
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	// Find joiner player (not owner)
	var joinerPlayerID string
	for _, p := range players {
		pm := p.(map[string]any)
		if pm["displayName"] == "妈妈" {
			joinerPlayerID = pm["id"].(string)
			break
		}
	}
	if joinerPlayerID == "" {
		t.Fatal("could not find joiner player")
	}

	// Owner gives 5 to joiner via score transfer
	_ = postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{joinerPlayerID},
		"amount":            5,
		"idempotencyKey":    "api-test-1",
	})
	summary = getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/summary")
	if summary["scoreTransferCount"].(float64) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	ranking := getJSON[[]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/ranking")
	if len(ranking) != 2 {
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
	joinResult := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{
		"inviteCode": game["inviteCode"], "displayName": "妈妈",
	})
	if joinResult["gameSessionId"] != gameID {
		t.Fatalf("unexpected join result: %+v", joinResult)
	}

	// Get joiner's player ID from summary
	summary := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/summary")
	players := summary["players"].([]any)
	var joinerPlayerID string
	for _, p := range players {
		pm := p.(map[string]any)
		if pm["displayName"] == "妈妈" {
			joinerPlayerID = pm["id"].(string)
			break
		}
	}
	if joinerPlayerID == "" {
		t.Fatal("could not find joiner player")
	}

	// Owner gives 5 to joiner via score transfer (makes owner's score non-zero)
	postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{joinerPlayerID},
		"amount":            5,
		"idempotencyKey":    "leave-test-api",
	})

	// Try to leave with non-zero score - should fail with 409
	req, _ := http.NewRequest(http.MethodPost, "/api/game-sessions/"+gameID+"/leave", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 status, got %d body %s", rec.Code, rec.Body.String())
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

func postJSONExpectStatus(t *testing.T, router http.Handler, token, path string, payload any, want int) {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("POST %s status %d, want %d body %s", path, rec.Code, want, rec.Body.String())
	}
}

func patchJSONExpectStatus(t *testing.T, router http.Handler, token, path string, payload any, want int) {
	t.Helper()
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != want {
		t.Fatalf("PATCH %s status %d, want %d body %s", path, rec.Code, want, rec.Body.String())
	}
}

func TestGameScoringLifecycleE2E(t *testing.T) {
	app := newTestApp(t)
	tokens := jwtauth.NewJWTService("test-signing-key", 720*time.Hour)
	router := api.NewRouter(slog.Default(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
		Tokens: tokens, WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})

	// 1. Owner login.
	ownerToken := loginHTTP(t, router, "owner-code")

	// 2. Create game with capacity.
	game := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions", map[string]any{
		"name": "家庭聚会", "maxParticipants": 4,
	})
	gameID := game["id"].(string)
	inviteCode := game["inviteCode"].(string)

	// 3. Current game returns created game.
	current := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/current")
	if current["id"] != gameID {
		t.Fatalf("unexpected current game: %+v", current)
	}

	// 4. Joiner logs in.
	joinerToken := loginHTTP(t, router, "joiner-code")

	// 5. Join preview shows no score details.
	preview := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join-preview", map[string]any{
		"inviteCode": inviteCode,
	})
	if preview["participantCount"].(float64) != 1 {
		t.Fatalf("unexpected preview participant count: %+v", preview)
	}
	if preview["gameSessionId"] != gameID {
		t.Fatalf("unexpected preview game id: %+v", preview)
	}
	// Ensure score-related fields are absent from join preview.
	if _, ok := preview["totalScore"]; ok {
		t.Fatalf("join preview should not contain totalScore: %+v", preview)
	}
	if _, ok := preview["scoreTransferCount"]; ok {
		t.Fatalf("join preview should not contain scoreTransferCount: %+v", preview)
	}
	if _, ok := preview["transfers"]; ok {
		t.Fatalf("join preview should not contain transfers: %+v", preview)
	}

	// 6. Joiner confirms join.
	joined := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join", map[string]any{
		"inviteCode": inviteCode, "displayName": "妈妈",
	})
	if joined["gameSessionId"] != gameID {
		t.Fatalf("unexpected join: %+v", joined)
	}

	// Get joiner's player ID for score transfer
	joinerPlayer := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/join-preview", map[string]any{
		"inviteCode": inviteCode,
	})
	joinerParticipantID := joinerPlayer["participants"].([]any)[1].(map[string]any)["id"].(string)

	// 6b. Third participant logs in and joins.
	thirdToken := loginHTTP(t, router, "third-code")
	_ = postJSON[map[string]any](t, router, thirdToken, "/api/game-sessions/join", map[string]any{
		"inviteCode": inviteCode, "displayName": "爸爸",
	})

	// Get third participant's player ID.
	thirdPreview := postJSON[map[string]any](t, router, thirdToken, "/api/game-sessions/join-preview", map[string]any{
		"inviteCode": inviteCode,
	})
	thirdParticipantID := thirdPreview["participants"].([]any)[2].(map[string]any)["id"].(string)

	// 7. Owner submits multi-receiver score transfer to joiner and third participant.
	_ = postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{joinerParticipantID, thirdParticipantID},
		"amount":            10,
		"idempotencyKey":    "e2e-transfer-1",
	})

	// 8. Summary shows participants by join order (owner first, joiners after) and transfer details.
	summary := getJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/summary")
	participants := summary["players"].([]any)
	if len(participants) != 3 {
		t.Fatalf("unexpected participants: %+v", participants)
	}
	// Summary returns participants by join order: owner first, joiner second, third third.
	ownerSummary := participants[0].(map[string]any)
	joinerSummary := participants[1].(map[string]any)
	thirdSummary := participants[2].(map[string]any)
	if ownerSummary["totalScore"].(float64) != -20 {
		t.Fatalf("unexpected owner score: %+v", ownerSummary)
	}
	if joinerSummary["totalScore"].(float64) != 10 {
		t.Fatalf("unexpected joiner score: %+v", joinerSummary)
	}
	if thirdSummary["totalScore"].(float64) != 10 {
		t.Fatalf("unexpected third score: %+v", thirdSummary)
	}
	if summary["scoreTransferCount"].(float64) != 1 {
		t.Fatalf("unexpected score transfer count: %+v", summary)
	}

	// Check ranking: score desc, display name asc for ties.
	ranking := getJSON[[]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/ranking")
	if len(ranking) != 3 {
		t.Fatalf("unexpected ranking: %+v", ranking)
	}
	// Both receivers (+10) tie; display name ASC for tie-break.
	first := ranking[0].(map[string]any)
	if first["totalScore"].(float64) != 10 {
		t.Fatalf("unexpected first score: %+v", first)
	}
	second := ranking[1].(map[string]any)
	if second["totalScore"].(float64) != 10 {
		t.Fatalf("unexpected second score: %+v", second)
	}
	ownerRanking := ranking[2].(map[string]any)
	if ownerRanking["totalScore"].(float64) != -20 {
		t.Fatalf("unexpected owner ranking score: %+v", ownerRanking)
	}

	// 9. Non-owner creates finish request.
	finishReq := postJSON[map[string]any](t, router, joinerToken, "/api/game-sessions/"+gameID+"/finish-requests", map[string]any{})
	if finishReq["Status"] != "pending" {
		t.Fatalf("unexpected finish request: %+v", finishReq)
	}

	// 10. Owner approves.
	finished := postJSON[map[string]any](t, router, ownerToken, "/api/game-sessions/"+gameID+"/finish-requests/"+finishReq["ID"].(string)+"/approve", map[string]any{})
	if finished["status"] != "finished" {
		t.Fatalf("unexpected finished status: %+v", finished)
	}
	shareToken := finished["publicShareToken"].(string)
	if shareToken == "" {
		t.Fatalf("expected share token, got %+v", finished)
	}

	// 11. History lists finished game for participants.
	history := getJSON[map[string]any](t, router, ownerToken, "/api/history/game-sessions")
	items := history["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("unexpected history: %+v", history)
	}
	historyItem := items[0].(map[string]any)
	if historyItem["id"] != gameID {
		t.Fatalf("unexpected history item: %+v", historyItem)
	}

	// Joiner should also see history
	joinerHistory := getJSON[map[string]any](t, router, joinerToken, "/api/history/game-sessions")
	joinerItems := joinerHistory["items"].([]any)
	if len(joinerItems) != 1 {
		t.Fatalf("unexpected joiner history: %+v", joinerHistory)
	}

	// 12. Public settlement share works without auth.
	public := getJSON[map[string]any](t, router, "", "/api/public/settlements/"+shareToken)
	if public["gameSessionId"] != gameID {
		t.Fatalf("unexpected public share: %+v", public)
	}
	publicParticipants := public["participants"].([]any)
	if len(publicParticipants) != 3 {
		t.Fatalf("unexpected public participants: %+v", publicParticipants)
	}

	// 13. Further join/leave/score transfer/profile update all fail.
	postJSONExpectStatus(t, router, joinerToken, "/api/game-sessions/join", map[string]any{
		"inviteCode": inviteCode, "displayName": "孩子",
	}, http.StatusConflict)

	postJSONExpectStatus(t, router, joinerToken, "/api/game-sessions/"+gameID+"/leave", map[string]any{},
		http.StatusConflict)

	postJSONExpectStatus(t, router, ownerToken, "/api/game-sessions/"+gameID+"/score-transfers", map[string]any{
		"receiverPlayerIds": []string{joinerParticipantID},
		"amount":            5,
		"idempotencyKey":    "e2e-transfer-after-finish",
	}, http.StatusConflict)

	patchJSONExpectStatus(t, router, joinerToken, "/api/game-sessions/"+gameID+"/my-profile", map[string]any{
		"displayName": "新名字",
	}, http.StatusConflict)
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
