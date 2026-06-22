package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"go.uber.org/zap"
)

type capturedWechatLog struct {
	level   string
	message string
	fields  map[string]any
}

type capturedWechatLogger struct {
	entries []capturedWechatLog
}

func (l *capturedWechatLogger) Info(message string, fields ...zap.Field)  { l.entries = append(l.entries, newCapturedWechatLog("info", message, fields...)) }
func (l *capturedWechatLogger) Debug(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedWechatLog("debug", message, fields...)) }
func (l *capturedWechatLogger) Error(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedWechatLog("error", message, fields...)) }
func (l *capturedWechatLogger) Warn(message string, fields ...zap.Field)  { l.entries = append(l.entries, newCapturedWechatLog("warn", message, fields...)) }
func (l *capturedWechatLogger) Fatal(message string, fields ...zap.Field) { l.entries = append(l.entries, newCapturedWechatLog("fatal", message, fields...)) }
func (l *capturedWechatLogger) InfoF(string, ...any)                       {}
func (l *capturedWechatLogger) DebugF(string, ...any)                      {}
func (l *capturedWechatLogger) ErrorF(string, ...any)                      {}
func (l *capturedWechatLogger) WarnF(string, ...any)                       {}
func (l *capturedWechatLogger) FatalF(string, ...any)                      {}
func (l *capturedWechatLogger) Sync() error                                { return nil }

func newCapturedWechatLog(level, message string, fields ...zap.Field) capturedWechatLog {
	entry := capturedWechatLog{
		level:   level,
		message: message,
		fields:  map[string]any{},
	}
	for _, field := range fields {
		if field.Interface != nil {
			entry.fields[field.Key] = field.Interface
			continue
		}
		if field.String != "" {
			entry.fields[field.Key] = field.String
			continue
		}
		entry.fields[field.Key] = field.Integer
	}
	return entry
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(status int, body any) *http.Response {
	payload, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(payload)),
	}
}

func byteResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestHTTPClientCodeToSessionCallsJscode2Session(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/sns/jscode2session" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("appid") != "wx-test-app" {
			t.Fatalf("unexpected appid %q", query.Get("appid"))
		}
		if query.Get("secret") != "test-secret" {
			t.Fatalf("unexpected secret %q", query.Get("secret"))
		}
		if query.Get("js_code") != "login-code" {
			t.Fatalf("unexpected js_code %q", query.Get("js_code"))
		}
		if query.Get("grant_type") != "authorization_code" {
			t.Fatalf("unexpected grant_type %q", query.Get("grant_type"))
		}
		return jsonResponse(http.StatusOK, map[string]string{
			"openid":  "openid-123",
			"unionid": "unionid-456",
		}), nil
	})}

	wechatClient := NewHTTPClient("wx-test-app", "test-secret", "https://wechat.test", client, nil)
	session, err := wechatClient.CodeToSession(context.Background(), "login-code")
	if err != nil {
		t.Fatalf("CodeToSession returned error: %v", err)
	}
	if session.OpenID != "openid-123" {
		t.Fatalf("unexpected openid %q", session.OpenID)
	}
	if session.UnionID == nil || *session.UnionID != "unionid-456" {
		t.Fatalf("unexpected unionid %#v", session.UnionID)
	}
}

func TestHTTPClientCodeToSessionRejectsWechatError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{
			"errcode": 40029,
			"errmsg":  "invalid code",
		}), nil
	})}

	wechatClient := NewHTTPClient("wx-test-app", "test-secret", "https://wechat.test", client, nil)
	if _, err := wechatClient.CodeToSession(context.Background(), "bad-code"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHTTPClientCodeToSessionLogsFailureContext(t *testing.T) {
	log := &capturedWechatLogger{}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return byteResponse(http.StatusBadGateway, []byte("upstream failed")), nil
	})}

	wechatClient := NewHTTPClient("wx-test-app", "test-secret", "https://wechat.test", client, log)
	if _, err := wechatClient.CodeToSession(context.Background(), "bad-code"); err == nil {
		t.Fatal("expected error")
	}

	if len(log.entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(log.entries))
	}
	entry := log.entries[0]
	if entry.level != "error" {
		t.Fatalf("expected error log level, got %s", entry.level)
	}
	if entry.message != "wechat request failed" {
		t.Fatalf("unexpected log message %q", entry.message)
	}
	if got := entry.fields["operation"]; got != "code_to_session" {
		t.Fatalf("expected operation code_to_session, got %#v", got)
	}
}

func TestHTTPClientGetUnlimitedQRCodeCallsWechatEndpoints(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			query := r.URL.Query()
			if query.Get("grant_type") != "client_credential" {
				t.Fatalf("unexpected grant_type %q", query.Get("grant_type"))
			}
			if query.Get("appid") != "wx-test-app" {
				t.Fatalf("unexpected appid %q", query.Get("appid"))
			}
			if query.Get("secret") != "test-secret" {
				t.Fatalf("unexpected secret %q", query.Get("secret"))
			}
			return jsonResponse(http.StatusOK, map[string]string{"access_token": "access-token-123"}), nil
		case "/wxa/getwxacodeunlimit":
			if r.URL.Query().Get("access_token") != "access-token-123" {
				t.Fatalf("unexpected access token %q", r.URL.Query().Get("access_token"))
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["page"] != "pages/game-join/index" {
				t.Fatalf("unexpected page %#v", payload["page"])
			}
			if payload["scene"] != "code=ABC123" {
				t.Fatalf("unexpected scene %#v", payload["scene"])
			}
			return byteResponse(http.StatusOK, []byte("png-bytes")), nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return nil, nil
	})}

	wechatClient := NewHTTPClient("wx-test-app", "test-secret", "https://wechat.test", client, nil)
	image, err := wechatClient.GetUnlimitedQRCode(context.Background(), "pages/game-join/index", "code=ABC123")
	if err != nil {
		t.Fatalf("GetUnlimitedQRCode returned error: %v", err)
	}
	if string(image) != "png-bytes" {
		t.Fatalf("unexpected image bytes %q", string(image))
	}
}
