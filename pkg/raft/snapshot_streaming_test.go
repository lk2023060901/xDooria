package raft

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotWriterReader(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	t.Run("write and read single KV", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

		writer, err := NewSnapshotWriter(sink, cfg)
		require.NoError(t, err)

		err = writer.WriteKV("key1", []byte("value1"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Read back
		reader, err := NewSnapshotReader(bytes.NewReader(sink.buf.Bytes()), cfg)
		require.NoError(t, err)

		record, err := reader.ReadRecord()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeKV, record.Type)

		key, value, err := ParseKV(record)
		require.NoError(t, err)
		assert.Equal(t, "key1", key)
		assert.Equal(t, []byte("value1"), value)

		// Read EOF
		record, err = reader.ReadRecord()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeEOF, record.Type)
	})

	t.Run("write and read multiple KVs", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

		writer, err := NewSnapshotWriter(sink, cfg)
		require.NoError(t, err)

		testData := map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
			"key3": []byte("value3"),
		}

		for k, v := range testData {
			err = writer.WriteKV(k, v)
			require.NoError(t, err)
		}

		err = writer.Close()
		require.NoError(t, err)

		// Read back
		reader, err := NewSnapshotReader(bytes.NewReader(sink.buf.Bytes()), cfg)
		require.NoError(t, err)

		readData := make(map[string][]byte)
		for {
			record, err := reader.ReadRecord()
			require.NoError(t, err)

			if record.Type == MessageTypeEOF {
				break
			}

			key, value, err := ParseKV(record)
			require.NoError(t, err)
			readData[key] = value
		}

		assert.Equal(t, testData, readData)
	})

	t.Run("custom record", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

		writer, err := NewSnapshotWriter(sink, cfg)
		require.NoError(t, err)

		customData := []byte(`{"type":"service","name":"game-server"}`)
		err = writer.WriteRecord(MessageTypeCustom, customData)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Read back
		reader, err := NewSnapshotReader(bytes.NewReader(sink.buf.Bytes()), cfg)
		require.NoError(t, err)

		record, err := reader.ReadRecord()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeCustom, record.Type)
		assert.Equal(t, customData, record.Data)
	})
}

func TestSimpleStreamingSnapshot(t *testing.T) {
	testData := map[string][]byte{
		"user:1":    []byte(`{"id":1,"name":"alice"}`),
		"user:2":    []byte(`{"id":2,"name":"bob"}`),
		"config:db": []byte(`{"host":"localhost","port":5432}`),
	}

	cfg := DefaultSnapshotConfig()
	snapshot := NewSimpleStreamingSnapshot(testData, cfg)

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	err := snapshot.Persist(sink)
	require.NoError(t, err)
	assert.True(t, sink.closed)

	// Restore
	restored, err := RestoreStreamingSnapshot(bytes.NewReader(sink.buf.Bytes()), cfg)
	require.NoError(t, err)
	assert.Equal(t, testData, restored)
}

func TestSnapshotChecksum(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	cfg.EnableChecksum = true

	t.Run("valid checksum", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

		writer, err := NewSnapshotWriter(sink, cfg)
		require.NoError(t, err)

		err = writer.WriteKV("test", []byte("data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Read should succeed
		reader, err := NewSnapshotReader(bytes.NewReader(sink.buf.Bytes()), cfg)
		require.NoError(t, err)

		_, err = reader.ReadRecord() // KV
		require.NoError(t, err)
		_, err = reader.ReadRecord() // EOF with checksum verification
		require.NoError(t, err)
	})

	t.Run("corrupted checksum", func(t *testing.T) {
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

		writer, err := NewSnapshotWriter(sink, cfg)
		require.NoError(t, err)

		err = writer.WriteKV("test", []byte("data"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Corrupt checksum (last 4 bytes)
		data := sink.buf.Bytes()
		if len(data) >= 4 {
			data[len(data)-1] ^= 0xFF // Flip bits in checksum
		}

		// Read should fail with checksum error
		reader, err := NewSnapshotReader(bytes.NewReader(data), cfg)
		require.NoError(t, err)

		// Read records until we hit checksum verification at EOF
		for {
			_, err = reader.ReadRecord()
			if err != nil {
				assert.ErrorIs(t, err, ErrSnapshotCorrupted)
				break
			}
		}
	})
}

func TestSnapshotWithoutChecksum(t *testing.T) {
	cfg := &SnapshotConfig{
		EnableChecksum: false,
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

	writer, err := NewSnapshotWriter(sink, cfg)
	require.NoError(t, err)

	err = writer.WriteKV("key", []byte("value"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Snapshot should be smaller (no checksum)
	reader, err := NewSnapshotReader(bytes.NewReader(sink.buf.Bytes()), cfg)
	require.NoError(t, err)

	record, err := reader.ReadRecord()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeKV, record.Type)

	record, err = reader.ReadRecord()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeEOF, record.Type)
}

func TestSnapshotCancel(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

	writer, err := NewSnapshotWriter(sink, cfg)
	require.NoError(t, err)

	err = writer.WriteKV("key", []byte("value"))
	require.NoError(t, err)

	err = writer.Cancel()
	require.NoError(t, err)
	assert.True(t, sink.cancelled)
}

func TestParseKV_InvalidRecord(t *testing.T) {
	t.Run("wrong type", func(t *testing.T) {
		record := &SnapshotRecord{Type: MessageTypeCustom, Data: []byte("data")}
		_, _, err := ParseKV(record)
		require.Error(t, err)
	})

	t.Run("data too short", func(t *testing.T) {
		record := &SnapshotRecord{Type: MessageTypeKV, Data: []byte{1, 2}}
		_, _, err := ParseKV(record)
		require.Error(t, err)
	})
}

func TestEmptySnapshot(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}

	writer, err := NewSnapshotWriter(sink, cfg)
	require.NoError(t, err)

	// Write nothing, just close
	err = writer.Close()
	require.NoError(t, err)

	// Restore empty snapshot
	restored, err := RestoreStreamingSnapshot(bytes.NewReader(sink.buf.Bytes()), cfg)
	require.NoError(t, err)
	assert.Empty(t, restored)
}
