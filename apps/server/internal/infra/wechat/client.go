package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

const defaultAPIBaseURL = "https://api.weixin.qq.com"

type Session struct {
	OpenID  string
	UnionID *string
}

type Client interface {
	CodeToSession(ctx context.Context, code string) (Session, error)
}

type HTTPClient struct {
	appID     string
	appSecret string
	baseURL   string
	client    *http.Client
}

func NewHTTPClient(appID, appSecret, baseURL string, client *http.Client) *HTTPClient {
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPClient{
		appID:     strings.TrimSpace(appID),
		appSecret: strings.TrimSpace(appSecret),
		baseURL:   strings.TrimRight(baseURL, "/"),
		client:    client,
	}
}

func (c *HTTPClient) CodeToSession(ctx context.Context, code string) (Session, error) {
	code = strings.TrimSpace(code)
	if c.appID == "" || c.appSecret == "" || code == "" {
		return Session{}, domain.ErrInvalidArgument
	}

	endpoint, err := url.Parse(c.baseURL + "/sns/jscode2session")
	if err != nil {
		return Session{}, err
	}
	query := endpoint.Query()
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Session{}, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return Session{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Session{}, fmt.Errorf("wechat jscode2session status %d", res.StatusCode)
	}

	var body struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return Session{}, err
	}
	if body.ErrCode != 0 {
		return Session{}, fmt.Errorf("%w: wechat jscode2session error %d: %s", domain.ErrUnauthorized, body.ErrCode, body.ErrMsg)
	}
	if body.OpenID == "" {
		return Session{}, errors.New("wechat jscode2session missing openid")
	}
	var unionID *string
	if body.UnionID != "" {
		unionID = &body.UnionID
	}
	return Session{OpenID: body.OpenID, UnionID: unionID}, nil
}
