package kafka

import (
	"testing"
	"time"
)

func TestConsumerStateString(t *testing.T) {
	tests := []struct {
		state    ConsumerState
		expected string
	}{
		{ConsumerStateIdle, "idle"},
		{ConsumerStateRunning, "running"},
		{ConsumerStateStopping, "stopping"},
		{ConsumerStateStopped, "stopped"},
		{ConsumerState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("ConsumerState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMessageStruct(t *testing.T) {
	now := time.Now()
	msg := &Message{
		Topic:     "test-topic",
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
		Headers:   map[string]string{"header1": "value1"},
		Partition: 0,
		Offset:    100,
		Timestamp: now,
	}

	if msg.Topic != "test-topic" {
		t.Errorf("expected Topic to be test-topic, got %s", msg.Topic)
	}
	if string(msg.Key) != "test-key" {
		t.Errorf("expected Key to be test-key, got %s", string(msg.Key))
	}
	if string(msg.Value) != "test-value" {
		t.Errorf("expected Value to be test-value, got %s", string(msg.Value))
	}
	if msg.Headers["header1"] != "value1" {
		t.Errorf("expected header1 to be value1, got %s", msg.Headers["header1"])
	}
	if msg.Partition != 0 {
		t.Errorf("expected Partition to be 0, got %d", msg.Partition)
	}
	if msg.Offset != 100 {
		t.Errorf("expected Offset to be 100, got %d", msg.Offset)
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp to be %v, got %v", now, msg.Timestamp)
	}
}

func TestPublishResult(t *testing.T) {
	result := &PublishResult{
		Topic:     "test-topic",
		Partition: 1,
		Offset:    200,
		Error:     nil,
	}

	if result.Topic != "test-topic" {
		t.Errorf("expected Topic to be test-topic, got %s", result.Topic)
	}
	if result.Partition != 1 {
		t.Errorf("expected Partition to be 1, got %d", result.Partition)
	}
	if result.Offset != 200 {
		t.Errorf("expected Offset to be 200, got %d", result.Offset)
	}
	if result.Error != nil {
		t.Errorf("expected Error to be nil, got %v", result.Error)
	}
}

func TestTopicPartition(t *testing.T) {
	tp := TopicPartition{
		Topic:     "test-topic",
		Partition: 3,
	}

	if tp.Topic != "test-topic" {
		t.Errorf("expected Topic to be test-topic, got %s", tp.Topic)
	}
	if tp.Partition != 3 {
		t.Errorf("expected Partition to be 3, got %d", tp.Partition)
	}
}

func TestPartitionOffset(t *testing.T) {
	po := PartitionOffset{
		Topic:     "test-topic",
		Partition: 2,
		Offset:    500,
	}

	if po.Topic != "test-topic" {
		t.Errorf("expected Topic to be test-topic, got %s", po.Topic)
	}
	if po.Partition != 2 {
		t.Errorf("expected Partition to be 2, got %d", po.Partition)
	}
	if po.Offset != 500 {
		t.Errorf("expected Offset to be 500, got %d", po.Offset)
	}
}

func TestConsumerStats(t *testing.T) {
	now := time.Now()
	stats := ConsumerStats{
		MessagesConsumed:  100,
		MessagesSucceeded: 95,
		MessagesFailed:    5,
		LastMessageTime:   now,
	}

	if stats.MessagesConsumed != 100 {
		t.Errorf("expected MessagesConsumed to be 100, got %d", stats.MessagesConsumed)
	}
	if stats.MessagesSucceeded != 95 {
		t.Errorf("expected MessagesSucceeded to be 95, got %d", stats.MessagesSucceeded)
	}
	if stats.MessagesFailed != 5 {
		t.Errorf("expected MessagesFailed to be 5, got %d", stats.MessagesFailed)
	}
}

func TestProducerStats(t *testing.T) {
	now := time.Now()
	stats := ProducerStats{
		MessagesProduced:  200,
		MessagesSucceeded: 198,
		MessagesFailed:    2,
		LastMessageTime:   now,
	}

	if stats.MessagesProduced != 200 {
		t.Errorf("expected MessagesProduced to be 200, got %d", stats.MessagesProduced)
	}
	if stats.MessagesSucceeded != 198 {
		t.Errorf("expected MessagesSucceeded to be 198, got %d", stats.MessagesSucceeded)
	}
	if stats.MessagesFailed != 2 {
		t.Errorf("expected MessagesFailed to be 2, got %d", stats.MessagesFailed)
	}
}
