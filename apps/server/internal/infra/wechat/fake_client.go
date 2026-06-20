package wechat

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/xuanye/one-round/apps/server/internal/domain"
)

type FakeClient struct{}

func (FakeClient) CodeToSession(_ context.Context, code string) (Session, error) {
	if strings.TrimSpace(code) == "" {
		return Session{}, domain.ErrInvalidArgument
	}
	sum := sha256.Sum256([]byte(code))
	return Session{OpenID: "fake_" + hex.EncodeToString(sum[:])[:24]}, nil
}

func (FakeClient) GetUnlimitedQRCode(_ context.Context, page string, scene string) ([]byte, error) {
	if strings.TrimSpace(page) == "" || strings.TrimSpace(scene) == "" {
		return nil, domain.ErrInvalidArgument
	}

	// 1x1 PNG for tests and fake-auth local mode.
	return base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jXioAAAAASUVORK5CYII=")
}
