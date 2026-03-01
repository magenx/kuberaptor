package config

import (
	"testing"
)

func TestDNSZoneSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		zone     DNSZone
		expected DNSZone
	}{
		{
			name: "Default TTL",
			zone: DNSZone{
				Enabled: true,
			},
			expected: DNSZone{
				Enabled: true,
				TTL:     3600,
			},
		},
		{
			name: "Custom TTL",
			zone: DNSZone{
				Enabled: true,
				TTL:     7200,
			},
			expected: DNSZone{
				Enabled: true,
				TTL:     7200,
			},
		},
		{
			name: "Disabled zone",
			zone: DNSZone{
				Enabled: false,
			},
			expected: DNSZone{
				Enabled: false,
				TTL:     3600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := tt.zone
			zone.SetDefaults()

			if zone.TTL != tt.expected.TTL {
				t.Errorf("TTL = %d, expected %d", zone.TTL, tt.expected.TTL)
			}
		})
	}
}
