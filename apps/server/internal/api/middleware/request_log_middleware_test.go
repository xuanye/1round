package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xuanye/one-round/apps/server/internal/api/response"
	"github.com/xuanye/one-round/apps/server/internal/domain"
	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type capturedLogEntry struct {
	level   string
	message string
	fields  map[string]any
}

type capturedLogger struct {
	entries []capturedLogEntry
}

func (l *capturedLogger) Info(message string, fields ...zap.Field)  { l.entries = append(l.entries, newCapturedLogEntry("info", message, fields...)) }
func (l *capturedLogger) Debug(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedLogEntry("debug", message, fields...)) }
func (l *capturedLogger) Error(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedLogEntry("error", message, fields...)) }
func (l *capturedLogger) Warn(message string, fields ...zap.Field)  { l.entries = append(l.entries, newCapturedLogEntry("warn", message, fields...)) }
func (l *capturedLogger) Fatal(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedLogEntry("fatal", message, fields...)) }
func (l *capturedLogger) InfoF(message string, _ ...any)            {}
func (l *capturedLogger) DebugF(message string, _ ...any)           {}
func (l *capturedLogger) ErrorF(message string, _ ...any)           {}
func (l *capturedLogger) WarnF(message string, _ ...any)            {}
func (l *capturedLogger) FatalF(message string, _ ...any)           {}
func (l *capturedLogger) Sync() error                               { return nil }

func newCapturedLogEntry(level, message string, fields ...zap.Field) capturedLogEntry {
	entry := capturedLogEntry{
		level:   level,
		message: message,
		fields:  map[string]any{},
	}
	for _, field := range fields {
		entry.fields[field.Key] = zapFieldValue(field)
	}
	return entry
}

func zapFieldValue(field zap.Field) any {
	switch field.Type {
	case zapcore.StringType:
		return field.String
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return field.Integer
	case zapcore.DurationType:
		return field.Integer
	case zapcore.ErrorType:
		if field.Interface == nil {
			return nil
		}
		if err, ok := field.Interface.(error); ok {
			return err.Error()
		}
		return field.Interface
	default:
		if field.Interface != nil {
			return field.Interface
		}
		return field.String
	}
}

func TestRequestLogExpectedBusinessErrorLogsInfoWithContext(t *testing.T) {
	log := &capturedLogger{}
	req := httptest.NewRequest(http.MethodPost, "/api/game-sessions/game-123/score-transfers", nil)
	req = withRouteParams(req, map[string]string{"id": "game-123"})
	rec := httptest.NewRecorder()

	handler := RequestLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sw, ok := w.(interface{ SetUserID(string) }); ok {
			sw.SetUserID("user-123")
		}
		response.Error(w, domain.ErrDuplicateDisplayName)
	}))

	handler.ServeHTTP(rec, req)

	if len(log.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log.entries))
	}
	entry := log.entries[0]
	if entry.level != "info" {
		t.Fatalf("expected info log level, got %s", entry.level)
	}
	if got := entry.fields["user_id"]; got != "user-123" {
		t.Fatalf("expected user_id, got %#v", got)
	}
	if got := entry.fields["game_session_id"]; got != "game-123" {
		t.Fatalf("expected game_session_id, got %#v", got)
	}
	if got := entry.fields["error_code"]; got != int64(40901) {
		t.Fatalf("expected error_code 40901, got %#v", got)
	}
}

func TestRequestLogUnauthorizedLogsWarn(t *testing.T) {
	log := &capturedLogger{}
	req := httptest.NewRequest(http.MethodGet, "/api/game-sessions/current", nil)
	rec := httptest.NewRecorder()

	tokens := jwtauth.NewJWTService("test-signing-key", time.Hour)
	handler := RequestLog(log)(Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected auth middleware to stop request")
	})))

	handler.ServeHTTP(rec, req)

	if len(log.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log.entries))
	}
	if log.entries[0].level != "warn" {
		t.Fatalf("expected warn log level, got %s", log.entries[0].level)
	}
	if got := log.entries[0].fields["error_code"]; got != int64(40101) {
		t.Fatalf("expected error_code 40101, got %#v", got)
	}
}

func TestRequestLogInternalErrorLogsError(t *testing.T) {
	log := &capturedLogger{}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	handler := RequestLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, errors.New("boom"))
	}))

	handler.ServeHTTP(rec, req)

	if len(log.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log.entries))
	}
	if log.entries[0].level != "error" {
		t.Fatalf("expected error log level, got %s", log.entries[0].level)
	}
}

func withRouteParams(r *http.Request, params map[string]string) *http.Request {
	routeCtx := chi.NewRouteContext()
	for key, value := range params {
		routeCtx.URLParams.Add(key, value)
	}
	ctx := r.Context()
	ctx = context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
	return r.WithContext(ctx)
}
