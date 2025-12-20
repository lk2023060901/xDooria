package etcd

import (
	"testing"

	"go.etcd.io/etcd/api/v3/mvccpb"
)

func TestKVToKeyValue(t *testing.T) {
	etcdKV := &mvccpb.KeyValue{
		Key:            []byte("test-key"),
		Value:          []byte("test-value"),
		CreateRevision: 100,
		ModRevision:    200,
		Version:        5,
		Lease:          123,
	}

	kv := kvToKeyValue(etcdKV)

	if kv.Key != "test-key" {
		t.Errorf("Expected Key=test-key, got %s", kv.Key)
	}

	if string(kv.Value) != "test-value" {
		t.Errorf("Expected Value=test-value, got %s", string(kv.Value))
	}

	if kv.CreateRevision != 100 {
		t.Errorf("Expected CreateRevision=100, got %d", kv.CreateRevision)
	}

	if kv.ModRevision != 200 {
		t.Errorf("Expected ModRevision=200, got %d", kv.ModRevision)
	}

	if kv.Version != 5 {
		t.Errorf("Expected Version=5, got %d", kv.Version)
	}

	if kv.Lease != 123 {
		t.Errorf("Expected Lease=123, got %d", kv.Lease)
	}
}

func TestKeyValueFields(t *testing.T) {
	kv := &KeyValue{
		Key:            "my-key",
		Value:          []byte("my-value"),
		CreateRevision: 1,
		ModRevision:    2,
		Version:        1,
		Lease:          0,
	}

	if kv.Key == "" {
		t.Error("Key should not be empty")
	}

	if len(kv.Value) == 0 {
		t.Error("Value should not be empty")
	}

	if kv.CreateRevision <= 0 {
		t.Error("CreateRevision should be positive")
	}

	if kv.ModRevision <= 0 {
		t.Error("ModRevision should be positive")
	}

	if kv.Version <= 0 {
		t.Error("Version should be positive")
	}
}
