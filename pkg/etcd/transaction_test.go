package etcd

import (
	"testing"
)

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int64
	}{
		{
			name:     "zero",
			input:    []byte("0"),
			expected: 0,
		},
		{
			name:     "positive number",
			input:    []byte("12345"),
			expected: 12345,
		},
		{
			name:     "single digit",
			input:    []byte("7"),
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt64(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt64(%s) = %d, want %d", string(tt.input), result, tt.expected)
			}
		})
	}
}

func TestInt64ToString(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "zero",
			input:    0,
			expected: "0",
		},
		{
			name:     "positive number",
			input:    12345,
			expected: "12345",
		},
		{
			name:     "negative number",
			input:    -12345,
			expected: "-12345",
		},
		{
			name:     "single digit",
			input:    7,
			expected: "7",
		},
		{
			name:     "negative single digit",
			input:    -7,
			expected: "-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := int64ToString(tt.input)
			if result != tt.expected {
				t.Errorf("int64ToString(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInt64StringRoundTrip(t *testing.T) {
	tests := []int64{0, 1, 10, 100, 1000, 12345, -1, -10, -100, -12345}

	for _, n := range tests {
		str := int64ToString(n)
		if n >= 0 {
			parsed := parseInt64([]byte(str))
			if parsed != n {
				t.Errorf("Round trip failed for %d: got %d", n, parsed)
			}
		}
	}
}

func TestTxnResponse(t *testing.T) {
	resp := &TxnResponse{
		Succeeded: true,
		Responses: []interface{}{},
	}

	if !resp.Succeeded {
		t.Error("Expected Succeeded=true")
	}

	if resp.Responses == nil {
		t.Error("Responses should not be nil")
	}
}
