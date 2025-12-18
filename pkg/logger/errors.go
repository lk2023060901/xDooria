package logger

import "errors"

var (
	ErrInvalidOutputPath = errors.New("output path is required when file output is enabled")
	ErrNoOutputEnabled   = errors.New("at least one output (console or file) must be enabled")
)
