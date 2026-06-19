package wechat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientCodeToSessionCallsJscode2Session(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"openid":  "openid-123",
			"unionid": "unionid-456",
		})
	}))
	defer server.Close()

	client := NewHTTPClient("wx-test-app", "test-secret", server.URL, server.Client())
	session, err := client.CodeToSession(context.Background(), "login-code")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 40029,
			"errmsg":  "invalid code",
		})
	}))
	defer server.Close()

	client := NewHTTPClient("wx-test-app", "test-secret", server.URL, server.Client())
	if _, err := client.CodeToSession(context.Background(), "bad-code"); err == nil {
		t.Fatal("expected error")
	}
}
