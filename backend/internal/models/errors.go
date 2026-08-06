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
	ErrReportNotFound   = errors.New("report not found")

	ErrTextbookNotFound = errors.New("textbook not found")
	ErrUploadNotFound   = errors.New("upload not found")

	ErrCardTaskNotFound     = errors.New("card task not found")
	ErrCardNotFound         = errors.New("card not found")
	ErrCardProgressNotFound = errors.New("card progress not found")

	ErrPOINotFound = errors.New("poi not found")

	ErrPushSubscriptionNotFound = errors.New("push subscription not found")
)
