package auth

import (
	"context"
	"time"

	jwtauth "github.com/xuanye/one-round/apps/server/internal/infra/auth"
	"github.com/xuanye/one-round/apps/server/internal/infra/sqlite"
	"github.com/xuanye/one-round/apps/server/internal/infra/wechat"
)

type Service struct {
	queries *sqlite.Queries
	wechat  wechat.Client
	tokens  *jwtauth.JWTService
	now     func() time.Time
}

type LoginResult struct {
	Token string   `json:"token"`
	User  UserView `json:"user"`
}

type UserView struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
}

func NewService(q *sqlite.Queries, wc wechat.Client, tokens *jwtauth.JWTService, now func() time.Time) *Service {
	return &Service{queries: q, wechat: wc, tokens: tokens, now: now}
}

func (s *Service) LoginWithWechatCode(ctx context.Context, code string) (LoginResult, error) {
	session, err := s.wechat.CodeToSession(ctx, code)
	if err != nil {
		return LoginResult{}, err
	}
	user, err := s.queries.UpsertUserByOpenID(ctx, session.OpenID, s.now())
	if err != nil {
		return LoginResult{}, err
	}
	token, err := s.tokens.Issue(user.ID, s.now())
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token: token,
		User:  UserView{ID: user.ID, DisplayName: user.DisplayName, AvatarURL: user.AvatarURL},
	}, nil
}
