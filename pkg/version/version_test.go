package version

import "testing"

func TestGet(t *testing.T) {
	// Save original version
	originalVersion := Version
	defer func() { Version = originalVersion }()

	testCases := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "default version",
			version:  "dev",
			expected: "dev",
		},
		{
			name:     "custom version",
			version:  "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "version with prefix",
			version:  "v2.4.5",
			expected: "v2.4.5",
		},
		{
			name:     "dev version with commit",
			version:  "dev-abc1234",
			expected: "dev-abc1234",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Version = tc.version
			result := Get()
			if result != tc.expected {
				t.Errorf("expected version '%s', got '%s'", tc.expected, result)
			}
		})
	}
}
