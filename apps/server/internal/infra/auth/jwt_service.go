package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redhu/one-round/apps/server/internal/domain"
)

type JWTService struct {
	signingKey []byte
	ttl        time.Duration
}

type Claims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}

func NewJWTService(signingKey string, ttl time.Duration) *JWTService {
	return &JWTService{signingKey: []byte(signingKey), ttl: ttl}
}

func (s *JWTService) Issue(userID string, now time.Time) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.signingKey)
}

func (s *JWTService) Verify(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(*jwt.Token) (any, error) {
		return s.signingKey, nil
	})
	if err != nil || !token.Valid {
		return "", domain.ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || claims.UserID == "" {
		return "", domain.ErrUnauthorized
	}
	return claims.UserID, nil
}
