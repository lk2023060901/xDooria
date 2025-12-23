package raft

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotLeader(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNotLeader",
			err:  ErrNotLeader,
			want: true,
		},
		{
			name: "wrapped ErrNotLeader",
			err:  fmt.Errorf("operation failed: %w", ErrNotLeader),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsNotLeader(tt.err))
		})
	}
}

func TestIsLeadershipLost(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrLeadershipLost",
			err:  ErrLeadershipLost,
			want: true,
		},
		{
			name: "wrapped ErrLeadershipLost",
			err:  fmt.Errorf("apply failed: %w", ErrLeadershipLost),
			want: true,
		},
		{
			name: "other error",
			err:  ErrNotLeader,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsLeadershipLost(tt.err))
		})
	}
}

func TestIsNodeClosed(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNodeClosed",
			err:  ErrNodeClosed,
			want: true,
		},
		{
			name: "wrapped ErrNodeClosed",
			err:  fmt.Errorf("node error: %w", ErrNodeClosed),
			want: true,
		},
		{
			name: "other error",
			err:  ErrNotLeader,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsNodeClosed(tt.err))
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "ErrNotLeader is retryable",
			err:  ErrNotLeader,
			want: true,
		},
		{
			name: "ErrLeadershipLost is retryable",
			err:  ErrLeadershipLost,
			want: true,
		},
		{
			name: "ErrApplyTimeout is retryable",
			err:  ErrApplyTimeout,
			want: true,
		},
		{
			name: "wrapped ErrNotLeader is retryable",
			err:  fmt.Errorf("failed: %w", ErrNotLeader),
			want: true,
		},
		{
			name: "ErrNodeClosed is not retryable",
			err:  ErrNodeClosed,
			want: false,
		},
		{
			name: "ErrInvalidConfig is not retryable",
			err:  ErrInvalidConfig,
			want: false,
		},
		{
			name: "ErrSnapshotFailed is not retryable",
			err:  ErrSnapshotFailed,
			want: false,
		},
		{
			name: "random error is not retryable",
			err:  errors.New("random error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsRetryable(tt.err))
		})
	}
}

func TestErrorMessages(t *testing.T) {
	// Ensure all errors have meaningful messages
	errs := []error{
		ErrNotLeader,
		ErrLeadershipLost,
		ErrNodeClosed,
		ErrNodeNotReady,
		ErrApplyTimeout,
		ErrInvalidConfig,
		ErrNoLeader,
		ErrSnapshotFailed,
		ErrRestoreFailed,
		ErrInvalidCommand,
		ErrAlreadyBootstrapped,
		ErrMembershipChangePending,
		ErrServerNotFound,
	}

	for _, err := range errs {
		t.Run(err.Error(), func(t *testing.T) {
			assert.NotEmpty(t, err.Error())
			assert.Contains(t, err.Error(), "raft:")
		})
	}
}
