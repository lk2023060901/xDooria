package session

import "errors"

var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrConnectionClosed = errors.New("connection closed")
	ErrBroadcastFailed  = errors.New("broadcast partially failed")
	ErrCloseFailed      = errors.New("close partially failed")
)
