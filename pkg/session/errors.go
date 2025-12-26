package session

import "errors"

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrConnectionClosed = errors.New("connection closed")
)
