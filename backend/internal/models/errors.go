package models

import "errors"

// Доменные ошибки - общие для repository и services
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrNicknameExists     = errors.New("nickname already exists")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrTokenHashExists    = errors.New("token hash already exists")

	ErrThreadNotFound   = errors.New("thread not found")
	ErrCommentNotFound  = errors.New("comment not found")
	ErrReactionNotFound = errors.New("reaction not found")
)
