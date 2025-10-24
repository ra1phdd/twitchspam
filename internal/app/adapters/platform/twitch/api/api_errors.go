package api

import "errors"

var (
	ErrUserAuthNotCompleted = errors.New("user auth failed")
	ErrBadRequest           = errors.New("bad request")
	ErrRateLimited          = errors.New("rate limited")
	ErrNotFound             = errors.New("not found")
)
