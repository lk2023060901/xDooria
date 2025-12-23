package raft

import (
	"bytes"
	"io"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_EncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		cmd  *Command
	}{
		{
			name: "set command",
			cmd: &Command{
				Type: CommandTypeSet,
				Key:  "test-key",
				Data: []byte("test-value"),
			},
		},
		{
			name: "delete command",
			cmd: &Command{
				Type: CommandTypeDelete,
				Key:  "test-key",
			},
		},
		{
			name: "custom command",
			cmd: &Command{
				Type: CommandTypeCustom,
				Data: []byte(`{"action":"register","service":"game-server"}`),
			},
		},
		{
			name: "empty data",
			cmd: &Command{
				Type: CommandTypeSet,
				Key:  "empty-key",
				Data: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			data, err := EncodeCommand(tt.cmd)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Decode
			decoded, err := DecodeCommand(data)
			require.NoError(t, err)
			assert.Equal(t, tt.cmd.Type, decoded.Type)
			assert.Equal(t, tt.cmd.Key, decoded.Key)
			assert.Equal(t, tt.cmd.Data, decoded.Data)
		})
	}
}

func TestDecodeCommand_InvalidJSON(t *testing.T) {
	_, err := DecodeCommand([]byte("invalid json"))
	require.Error(t, err)
}

func TestApplyResult(t *testing.T) {
	t.Run("success result", func(t *testing.T) {
		result := NewApplyResult("success-data", nil)
		assert.Equal(t, "success-data", result.Data)
		assert.NoError(t, result.Error)
	})

	t.Run("error result", func(t *testing.T) {
		expectedErr := assert.AnError
		result := NewApplyResult(nil, expectedErr)
		assert.Nil(t, result.Data)
		assert.Equal(t, expectedErr, result.Error)
	})
}

func TestSimpleFSMSnapshot(t *testing.T) {
	snapshotData := []byte(`{"key1":"value1","key2":"value2"}`)
	snapshot := NewSimpleFSMSnapshot(snapshotData)

	t.Run("persist success", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		err := snapshot.Persist(sink)
		require.NoError(t, err)
		assert.Equal(t, snapshotData, sink.buf.Bytes())
		assert.True(t, sink.closed)
	})

	t.Run("persist write error", func(t *testing.T) {
		sink := &mockSnapshotSink{
			buf:      &bytes.Buffer{},
			writeErr: assert.AnError,
		}
		err := snapshot.Persist(sink)
		require.Error(t, err)
		assert.True(t, sink.cancelled)
	})

	t.Run("release is no-op", func(t *testing.T) {
		snapshot.Release() // Should not panic
	})
}

// mockSnapshotSink implements raft.SnapshotSink for testing
type mockSnapshotSink struct {
	buf       *bytes.Buffer
	writeErr  error
	closed    bool
	cancelled bool
}

func (m *mockSnapshotSink) Write(p []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.buf.Write(p)
}

func (m *mockSnapshotSink) Close() error {
	m.closed = true
	return nil
}

func (m *mockSnapshotSink) ID() string {
	return "test-snapshot-id"
}

func (m *mockSnapshotSink) Cancel() error {
	m.cancelled = true
	return nil
}

// mockFSM implements FSM for testing
type mockFSM struct {
	applied   [][]byte
	snapshots int
	restores  int
}

func newMockFSM() *mockFSM {
	return &mockFSM{
		applied: make([][]byte, 0),
	}
}

func (f *mockFSM) Apply(log *raft.Log) interface{} {
	f.applied = append(f.applied, log.Data)
	return NewApplyResult(len(f.applied), nil)
}

func (f *mockFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.snapshots++
	return NewSimpleFSMSnapshot([]byte("snapshot-data")), nil
}

func (f *mockFSM) Restore(snapshot io.ReadCloser) error {
	f.restores++
	defer snapshot.Close()
	return nil
}

func TestFSMWrapper(t *testing.T) {
	fsm := newMockFSM()
	wrapper := newFSMWrapper(fsm)

	t.Run("apply delegates to fsm", func(t *testing.T) {
		log := &raft.Log{Data: []byte("test-data")}
		result := wrapper.Apply(log)

		assert.Len(t, fsm.applied, 1)
		assert.Equal(t, []byte("test-data"), fsm.applied[0])

		applyResult, ok := result.(*ApplyResult)
		require.True(t, ok)
		assert.Equal(t, 1, applyResult.Data)
	})

	t.Run("snapshot delegates to fsm", func(t *testing.T) {
		snap, err := wrapper.Snapshot()
		require.NoError(t, err)
		assert.NotNil(t, snap)
		assert.Equal(t, 1, fsm.snapshots)
	})

	t.Run("restore delegates to fsm", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader([]byte("restore-data")))
		err := wrapper.Restore(reader)
		require.NoError(t, err)
		assert.Equal(t, 1, fsm.restores)
	})
}

func TestChunkingFSMWrapper(t *testing.T) {
	fsm := newMockFSM()
	wrapper := newFSMWrapper(fsm)
	chunkingWrapper := newChunkingFSMWrapper(wrapper)

	t.Run("apply delegates through chunking wrapper", func(t *testing.T) {
		log := &raft.Log{Data: []byte("chunked-data")}
		result := chunkingWrapper.Apply(log)

		// The result should be from the underlying FSM
		assert.Len(t, fsm.applied, 1)
		assert.Equal(t, []byte("chunked-data"), fsm.applied[0])

		applyResult, ok := result.(*ApplyResult)
		require.True(t, ok)
		assert.Equal(t, 1, applyResult.Data)
	})

	t.Run("snapshot delegates through chunking wrapper", func(t *testing.T) {
		snap, err := chunkingWrapper.Snapshot()
		require.NoError(t, err)
		assert.NotNil(t, snap)
		assert.Equal(t, 1, fsm.snapshots)
	})

	t.Run("restore delegates through chunking wrapper", func(t *testing.T) {
		reader := io.NopCloser(bytes.NewReader([]byte("restore-data")))
		err := chunkingWrapper.Restore(reader)
		require.NoError(t, err)
		assert.Equal(t, 1, fsm.restores)
	})

	t.Run("current state returns valid state", func(t *testing.T) {
		state, err := chunkingWrapper.CurrentState()
		require.NoError(t, err)
		// State can be nil if no chunking operations are in progress
		// This is expected behavior
		_ = state
	})

	t.Run("restore state accepts nil", func(t *testing.T) {
		err := chunkingWrapper.RestoreState(nil)
		require.NoError(t, err)
	})
}

