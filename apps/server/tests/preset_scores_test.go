package tests

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/xuanye/one-round/apps/server/internal/api"
	wshandler "github.com/xuanye/one-round/apps/server/internal/api/handler"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
)

func TestPresetScoresValidationDoesNotCreateGame(t *testing.T) {
	app := newTestApp(t)
	defer app.db.Close()
	ctx := context.Background()
	user := login(t, app, "owner")
	for _, scores := range [][]int{{}, {0}, {-1}, {100000}, {20, 20}, {1, 2, 3, 4, 5, 6, 7, 8, 9}} {
		if _, err := app.game.Create(ctx, user, "朋友局", nil, scores); !errors.Is(err, domain.ErrInvalidArgument) {
			t.Fatalf("scores %v: expected invalid argument, got %v", scores, err)
		}
	}
	current, err := app.game.Current(ctx, user)
	if err != nil || current != nil {
		t.Fatalf("invalid request created game: %+v, %v", current, err)
	}
	game, err := app.game.Create(ctx, user, "朋友局", nil, []int{1, 99999})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := app.q.GetGameSession(ctx, game.ID)
	if err != nil || !reflect.DeepEqual(stored.PresetScores, []int{1, 99999}) {
		t.Fatalf("stored scores: %+v, %v", stored, err)
	}
}

func TestPresetScoresAPISharedAcrossMembers(t *testing.T) {
	app := newTestApp(t)
	defer app.db.Close()
	router := api.NewRouter(logger.NewConsole(), api.Services{
		Auth: app.auth, Game: app.game, Player: app.player, ScoreTransfer: app.scoreTransfer, Settlement: app.settlement, Query: app.query,
		Tokens: jwtauth.NewJWTService("test-signing-key", 720*time.Hour), WebSocket: wshandler.NewWebSocketHandler(app.game, app.hub, 4, time.Second),
	})
	owner := loginHTTP(t, router, "owner")
	for _, scores := range []any{[]int{}, []int{20, 20}, []int{0}, []any{1.5}, []any{"20"}} {
		postJSONExpectStatus(t, router, owner, "/api/game-sessions", map[string]any{"name": "朋友局", "presetScores": scores}, http.StatusBadRequest)
	}
	game := postJSON[domain.GameSession](t, router, owner, "/api/game-sessions", map[string]any{"name": "朋友局", "presetScores": []int{25, 35, 45, 65}})
	want := []int{25, 35, 45, 65}
	if game.MaxParticipants != nil || !reflect.DeepEqual(game.PresetScores, want) {
		t.Fatalf("created: %+v", game)
	}
	joiner := loginHTTP(t, router, "joiner")
	postJSON[map[string]any](t, router, joiner, "/api/game-sessions/join", map[string]any{"inviteCode": game.InviteCode, "displayName": "牌友"})
	for _, token := range []string{owner, joiner} {
		summary := getJSON[struct {
			PresetScores []int `json:"presetScores"`
		}](t, router, token, "/api/game-sessions/"+game.ID+"/summary")
		current := getJSON[struct {
			PresetScores []int `json:"presetScores"`
		}](t, router, token, "/api/game-sessions/current")
		if !reflect.DeepEqual(summary.PresetScores, want) || !reflect.DeepEqual(current.PresetScores, want) {
			t.Fatalf("member scores: summary %+v, current %+v", summary, current)
		}
	}
	legacyOwner := loginHTTP(t, router, "legacy-owner")
	legacy := postJSON[domain.GameSession](t, router, legacyOwner, "/api/game-sessions", map[string]any{"name": "旧客户端"})
	if !reflect.DeepEqual(legacy.PresetScores, []int{20, 30, 40, 60}) {
		t.Fatalf("default scores: %v", legacy.PresetScores)
	}
}

func TestPresetScoresMigrationBackfillsExistingGame(t *testing.T) {
	db, err := sqlite.Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "../migrations", 7); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users(id,open_id,created_at,updated_at) VALUES('owner','openid',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO game_sessions(id,name,invite_code,owner_user_id,status,zero_sum_required,created_at,updated_at) VALUES('old','旧牌局','ABCDEF','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(db, "../migrations"); err != nil {
		t.Fatal(err)
	}
	var encoded string
	if err := db.QueryRow(`SELECT preset_scores FROM game_sessions WHERE id='old'`).Scan(&encoded); err != nil {
		t.Fatal(err)
	}
	if encoded != "[20,30,40,60]" {
		t.Fatalf("legacy scores: %s", encoded)
	}
}
