package types

import "testing"

func TestIPToInt(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected int64
	}{
		{
			name:     "valid IP 192.168.1.1",
			ip:       "192.168.1.1",
			expected: 3232235777, // (192 << 24) | (168 << 16) | (1 << 8) | 1
		},
		{
			name:     "valid IP 10.0.0.1",
			ip:       "10.0.0.1",
			expected: 167772161, // (10 << 24) | 1
		},
		{
			name:     "valid IP 0.0.0.0",
			ip:       "0.0.0.0",
			expected: 0,
		},
		{
			name:     "valid IP 255.255.255.255",
			ip:       "255.255.255.255",
			expected: 4294967295, // Max IPv4
		},
		{
			name:     "valid IP 8.8.8.8",
			ip:       "8.8.8.8",
			expected: 134744072, // (8 << 24) | (8 << 16) | (8 << 8) | 8
		},
		{
			name:     "invalid - too many octets",
			ip:       "192.168.1.1.1",
			expected: 0,
		},
		{
			name:     "invalid - too few octets",
			ip:       "192.168.1",
			expected: 0,
		},
		{
			name:     "invalid - octet out of range",
			ip:       "256.1.1.1",
			expected: 0,
		},
		{
			name:     "invalid - negative octet",
			ip:       "192.168.-1.1",
			expected: 0,
		},
		{
			name:     "invalid - contains letters",
			ip:       "192.168.a.1",
			expected: 0,
		},
		{
			name:     "invalid - empty string",
			ip:       "",
			expected: 0,
		},
		{
			name:     "invalid - extra dots",
			ip:       "192..168.1.1",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IPToInt(tt.ip)
			if result != tt.expected {
				t.Errorf("IPToInt(%q) = %d, expected %d", tt.ip, result, tt.expected)
			}
		})
	}
}
