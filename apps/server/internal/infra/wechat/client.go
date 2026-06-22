package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/xuanye/one-round/apps/server/internal/domain"
	"github.com/xuanye/one-round/apps/server/internal/infra/logger"
	"go.uber.org/zap"
)

const defaultAPIBaseURL = "https://api.weixin.qq.com"

type Session struct {
	OpenID  string
	UnionID *string
}

type Client interface {
	CodeToSession(ctx context.Context, code string) (Session, error)
	GetUnlimitedQRCode(ctx context.Context, page string, scene string) ([]byte, error)
}

type HTTPClient struct {
	appID     string
	appSecret string
	baseURL   string
	client    *http.Client
	log       logger.Logger
}

func NewHTTPClient(appID, appSecret, baseURL string, client *http.Client, log logger.Logger) *HTTPClient {
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	if log == nil {
		log = logger.NewNop()
	}
	return &HTTPClient{
		appID:     strings.TrimSpace(appID),
		appSecret: strings.TrimSpace(appSecret),
		baseURL:   strings.TrimRight(baseURL, "/"),
		client:    client,
		log:       log,
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
		c.logRequestFailure("code_to_session", 0, err)
		return Session{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		err := fmt.Errorf("wechat jscode2session status %d", res.StatusCode)
		c.logRequestFailure("code_to_session", res.StatusCode, err)
		return Session{}, err
	}

	var body struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		c.logRequestFailure("code_to_session", res.StatusCode, err)
		return Session{}, err
	}
	if body.ErrCode != 0 {
		err := fmt.Errorf("%w: wechat jscode2session error %d: %s", domain.ErrExternalServiceFailed, body.ErrCode, body.ErrMsg)
		c.logRequestFailure("code_to_session", res.StatusCode, err, zap.Int("wechat_errcode", body.ErrCode), zap.String("wechat_errmsg", body.ErrMsg))
		return Session{}, err
	}
	if body.OpenID == "" {
		err := errors.New("wechat jscode2session missing openid")
		c.logRequestFailure("code_to_session", res.StatusCode, err)
		return Session{}, err
	}
	var unionID *string
	if body.UnionID != "" {
		unionID = &body.UnionID
	}
	return Session{OpenID: body.OpenID, UnionID: unionID}, nil
}

func (c *HTTPClient) GetUnlimitedQRCode(ctx context.Context, page string, scene string) ([]byte, error) {
	page = strings.TrimSpace(page)
	scene = strings.TrimSpace(scene)
	if c.appID == "" || c.appSecret == "" || page == "" || scene == "" {
		return nil, domain.ErrInvalidArgument
	}

	accessToken, err := c.fetchAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/wxa/getwxacodeunlimit?access_token=%s", c.baseURL, url.QueryEscape(accessToken))
	payload, err := json.Marshal(map[string]any{
		"page":       page,
		"scene":      scene,
		"check_path": false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		c.logRequestFailure("get_unlimited_qrcode", 0, err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.logRequestFailure("get_unlimited_qrcode", res.StatusCode, err)
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		err := fmt.Errorf("wechat getwxacodeunlimit status %d", res.StatusCode)
		c.logRequestFailure("get_unlimited_qrcode", res.StatusCode, err)
		return nil, err
	}
	if looksLikeWechatError(body) {
		var apiErr struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(body, &apiErr); err != nil {
			c.logRequestFailure("get_unlimited_qrcode", res.StatusCode, err)
			return nil, err
		}
		err := fmt.Errorf("%w: wechat getwxacodeunlimit error %d: %s", domain.ErrExternalServiceFailed, apiErr.ErrCode, apiErr.ErrMsg)
		c.logRequestFailure("get_unlimited_qrcode", res.StatusCode, err, zap.Int("wechat_errcode", apiErr.ErrCode), zap.String("wechat_errmsg", apiErr.ErrMsg))
		return nil, err
	}
	return body, nil
}

func (c *HTTPClient) fetchAccessToken(ctx context.Context) (string, error) {
	endpoint, err := url.Parse(c.baseURL + "/cgi-bin/token")
	if err != nil {
		return "", err
	}
	query := endpoint.Query()
	query.Set("grant_type", "client_credential")
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	res, err := c.client.Do(req)
	if err != nil {
		c.logRequestFailure("fetch_access_token", 0, err)
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		err := fmt.Errorf("wechat access token status %d", res.StatusCode)
		c.logRequestFailure("fetch_access_token", res.StatusCode, err)
		return "", err
	}

	var body struct {
		AccessToken string `json:"access_token"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		c.logRequestFailure("fetch_access_token", res.StatusCode, err)
		return "", err
	}
	if body.ErrCode != 0 {
		err := fmt.Errorf("%w: wechat access token error %d: %s", domain.ErrExternalServiceFailed, body.ErrCode, body.ErrMsg)
		c.logRequestFailure("fetch_access_token", res.StatusCode, err, zap.Int("wechat_errcode", body.ErrCode), zap.String("wechat_errmsg", body.ErrMsg))
		return "", err
	}
	if body.AccessToken == "" {
		err := errors.New("wechat access token missing access_token")
		c.logRequestFailure("fetch_access_token", res.StatusCode, err)
		return "", err
	}
	return body.AccessToken, nil
}

func (c *HTTPClient) logRequestFailure(operation string, statusCode int, err error, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("operation", operation),
		zap.Error(err),
	}
	if statusCode > 0 {
		baseFields = append(baseFields, zap.Int("status_code", statusCode))
	}
	baseFields = append(baseFields, fields...)
	c.log.Error("wechat request failed", baseFields...)
}

func looksLikeWechatError(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	return strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"errcode"`)
}
