package wechat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/redhu/one-round/apps/server/internal/domain"
)

type FakeClient struct{}

func (FakeClient) CodeToSession(_ context.Context, code string) (Session, error) {
	if strings.TrimSpace(code) == "" {
		return Session{}, domain.ErrInvalidArgument
	}
	sum := sha256.Sum256([]byte(code))
	return Session{OpenID: "fake_" + hex.EncodeToString(sum[:])[:24]}, nil
}
