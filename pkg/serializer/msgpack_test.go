package serializer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string `codec:"name"`
	Value int    `codec:"value"`
	Data  []byte `codec:"data"`
}

func TestEncodeDecode(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		original := &testStruct{
			Name:  "test",
			Value: 42,
			Data:  []byte("hello"),
		}

		data, err := Encode(original)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded testStruct
		err = Decode(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.Value, decoded.Value)
		assert.Equal(t, original.Data, decoded.Data)
	})

	t.Run("map", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": float64(123),
		}

		data, err := Encode(original)
		require.NoError(t, err)

		var decoded map[string]interface{}
		err = Decode(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original["key1"], decoded["key1"])
		assert.Equal(t, original["key2"], decoded["key2"])
	})

	t.Run("slice", func(t *testing.T) {
		original := []string{"a", "b", "c"}

		data, err := Encode(original)
		require.NoError(t, err)

		var decoded []string
		err = Decode(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original, decoded)
	})

	t.Run("primitive types", func(t *testing.T) {
		// int
		intData, err := Encode(42)
		require.NoError(t, err)
		var intVal int
		err = Decode(intData, &intVal)
		require.NoError(t, err)
		assert.Equal(t, 42, intVal)

		// string
		strData, err := Encode("hello")
		require.NoError(t, err)
		var strVal string
		err = Decode(strData, &strVal)
		require.NoError(t, err)
		assert.Equal(t, "hello", strVal)

		// bool
		boolData, err := Encode(true)
		require.NoError(t, err)
		var boolVal bool
		err = Decode(boolData, &boolVal)
		require.NoError(t, err)
		assert.True(t, boolVal)
	})
}

func TestEncodeWithSizeHint(t *testing.T) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	original := &testStruct{
		Name:  "large",
		Value: 999,
		Data:  largeData,
	}

	// Use size hint close to actual size
	data, err := EncodeWithSizeHint(original, 10100)
	require.NoError(t, err)

	var decoded testStruct
	err = Decode(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Data, decoded.Data)
}

func TestNewEncoderDecoder(t *testing.T) {
	original := &testStruct{
		Name:  "stream",
		Value: 100,
	}

	// Encode using NewEncoder
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.Encode(original)
	require.NoError(t, err)

	// Decode using NewDecoder
	var decoded testStruct
	dec := NewDecoder(&buf)
	err = dec.Decode(&decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Name, decoded.Name)
	assert.Equal(t, original.Value, decoded.Value)
}

func TestDecodeInvalidData(t *testing.T) {
	var decoded testStruct
	err := Decode([]byte("invalid msgpack data"), &decoded)
	require.Error(t, err)
}

func TestEncodeNil(t *testing.T) {
	data, err := Encode(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func BenchmarkEncode(b *testing.B) {
	data := &testStruct{
		Name:  "benchmark",
		Value: 12345,
		Data:  []byte("some test data for benchmarking"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encode(data)
	}
}

func BenchmarkDecode(b *testing.B) {
	original := &testStruct{
		Name:  "benchmark",
		Value: 12345,
		Data:  []byte("some test data for benchmarking"),
	}
	encoded, _ := Encode(original)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded testStruct
		_ = Decode(encoded, &decoded)
	}
}

func BenchmarkEncodeParallel(b *testing.B) {
	data := &testStruct{
		Name:  "benchmark",
		Value: 12345,
		Data:  []byte("some test data for benchmarking"),
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = Encode(data)
		}
	})
}
