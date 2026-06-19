package wechat

import "context"

type Session struct {
	OpenID  string
	UnionID *string
}

type Client interface {
	CodeToSession(ctx context.Context, code string) (Session, error)
}
