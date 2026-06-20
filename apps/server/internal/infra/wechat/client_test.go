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

func TestHTTPClientGetUnlimitedQRCodeCallsWechatEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "access-token-123"})
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
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("png-bytes"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewHTTPClient("wx-test-app", "test-secret", server.URL, server.Client())
	image, err := client.GetUnlimitedQRCode(context.Background(), "pages/game-join/index", "code=ABC123")
	if err != nil {
		t.Fatalf("GetUnlimitedQRCode returned error: %v", err)
	}
	if string(image) != "png-bytes" {
		t.Fatalf("unexpected image bytes %q", string(image))
	}
}
