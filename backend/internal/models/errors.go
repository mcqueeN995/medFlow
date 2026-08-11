package models

import "errors"

// Доменные ошибки - общие для repository и services
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrNicknameExists     = errors.New("nickname already exists")
	ErrLoginExists        = errors.New("login already exists")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrTokenHashExists    = errors.New("token hash already exists")

	ErrLoginChangeRequestNotFound = errors.New("login change request not found")
	ErrLoginChangeRequestExpired  = errors.New("login change request expired")

	ErrPasswordResetRequestNotFound = errors.New("password reset request not found")

	ErrThreadNotFound   = errors.New("thread not found")
	ErrCommentNotFound  = errors.New("comment not found")
	ErrReactionNotFound = errors.New("reaction not found")
	ErrReportNotFound   = errors.New("report not found")

	ErrTextbookNotFound = errors.New("textbook not found")
	ErrUploadNotFound   = errors.New("upload not found")

	ErrCardTaskNotFound     = errors.New("card task not found")
	ErrCardNotFound         = errors.New("card not found")
	ErrCardProgressNotFound = errors.New("card progress not found")
	ErrCardFavoriteNotFound = errors.New("card favorite not found")
	ErrCardRatingNotFound   = errors.New("card rating not found")

	ErrPOINotFound = errors.New("poi not found")

	ErrPushSubscriptionNotFound = errors.New("push subscription not found")
)
