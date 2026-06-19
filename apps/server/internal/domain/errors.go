package domain

import "errors"

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrNotFound             = errors.New("not found")
	ErrGameSessionFinished  = errors.New("game session finished")
	ErrScoreTotalMustBeZero = errors.New("score total must be zero")
	ErrInvalidPlayer        = errors.New("invalid player")
	ErrPlayerAlreadyExists  = errors.New("player already exists")
	ErrGameMemberRequired   = errors.New("game member required")
	ErrConflict             = errors.New("conflict")
)
