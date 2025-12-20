package etcd

import (
	"testing"
	"time"
)

func TestWatchEventType(t *testing.T) {
	if EventTypePut != 0 {
		t.Errorf("Expected EventTypePut=0, got %d", EventTypePut)
	}

	if EventTypeDelete != 1 {
		t.Errorf("Expected EventTypeDelete=1, got %d", EventTypeDelete)
	}
}

func TestWatchEvent(t *testing.T) {
	event := &WatchEvent{
		Type:     EventTypePut,
		Key:      "test-key",
		Value:    []byte("test-value"),
		Revision: 100,
	}

	if event.Type != EventTypePut {
		t.Errorf("Expected Type=EventTypePut, got %d", event.Type)
	}

	if event.Key != "test-key" {
		t.Errorf("Expected Key=test-key, got %s", event.Key)
	}

	if string(event.Value) != "test-value" {
		t.Errorf("Expected Value=test-value, got %s", string(event.Value))
	}

	if event.Revision != 100 {
		t.Errorf("Expected Revision=100, got %d", event.Revision)
	}
}

func TestLeaseID(t *testing.T) {
	var id LeaseID = 12345

	if id != 12345 {
		t.Errorf("Expected LeaseID=12345, got %d", id)
	}
}

func TestElectionValue(t *testing.T) {
	ev := &ElectionValue{
		Key:      "election-key",
		Value:    []byte("candidate-value"),
		Revision: 200,
		LeaseID:  123,
	}

	if ev.Key != "election-key" {
		t.Errorf("Expected Key=election-key, got %s", ev.Key)
	}

	if string(ev.Value) != "candidate-value" {
		t.Errorf("Expected Value=candidate-value, got %s", string(ev.Value))
	}

	if ev.Revision != 200 {
		t.Errorf("Expected Revision=200, got %d", ev.Revision)
	}

	if ev.LeaseID != 123 {
		t.Errorf("Expected LeaseID=123, got %d", ev.LeaseID)
	}
}

func TestCompareTarget(t *testing.T) {
	targets := []CompareTarget{
		CompareVersion,
		CompareCreate,
		CompareMod,
		CompareValue,
		CompareLease,
	}

	for i, target := range targets {
		if int(target) != i {
			t.Errorf("Expected CompareTarget=%d, got %d", i, target)
		}
	}
}

func TestCompareResult(t *testing.T) {
	results := []CompareResult{
		CompareEqual,
		CompareGreater,
		CompareLess,
		CompareNotEqual,
	}

	for i, result := range results {
		if int(result) != i {
			t.Errorf("Expected CompareResult=%d, got %d", i, result)
		}
	}
}

func TestOpType(t *testing.T) {
	types := []OpType{
		OpTypeGet,
		OpTypePut,
		OpTypeDelete,
		OpTypeTxn,
	}

	for i, opType := range types {
		if int(opType) != i {
			t.Errorf("Expected OpType=%d, got %d", i, opType)
		}
	}
}

func TestLockOptions(t *testing.T) {
	opts := &lockOptions{
		ttl:     30,
		timeout: 5 * time.Second,
	}

	if opts.ttl != 30 {
		t.Errorf("Expected ttl=30, got %d", opts.ttl)
	}

	if opts.timeout != 5*time.Second {
		t.Errorf("Expected timeout=5s, got %v", opts.timeout)
	}
}

func TestElectionOptions(t *testing.T) {
	opts := &electionOptions{
		ttl: 60,
	}

	if opts.ttl != 60 {
		t.Errorf("Expected ttl=60, got %d", opts.ttl)
	}
}

func TestWatchOptions(t *testing.T) {
	opts := &watchOptions{
		revision:      100,
		prevKV:        true,
		prefix:        true,
		createdNotify: false,
	}

	if opts.revision != 100 {
		t.Errorf("Expected revision=100, got %d", opts.revision)
	}

	if !opts.prevKV {
		t.Error("Expected prevKV=true")
	}

	if !opts.prefix {
		t.Error("Expected prefix=true")
	}

	if opts.createdNotify {
		t.Error("Expected createdNotify=false")
	}
}
