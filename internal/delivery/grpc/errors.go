package grpcserver

import "errors"

var (
	ErrRateLimiterNotConfigured = errors.New("rate limiter not configured")
	ErrInvalidIP                = errors.New("invalid IP address")
	ErrSubnetListNotConfigured  = errors.New("subnet list not configured")
)
