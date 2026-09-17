package copybridge

import (
	"testing"
)

func TestCORSAllow(t *testing.T) {
	cases := []struct {
		name       string
		origin     string
		configured []string
		want       string
	}{
		{name: "no origin", origin: "", want: ""},
		{name: "localhost always allowed", origin: "http://localhost:3000", want: "http://localhost:3000"},
		{name: "loopback ip allowed", origin: "http://127.0.0.1:5173", want: "http://127.0.0.1:5173"},
		{name: "lan ip allowed by default", origin: "http://192.168.1.10:8080", want: "http://192.168.1.10:8080"},
		{name: "public origin blocked by default", origin: "https://example.com", want: ""},
		{name: "explicit config allows public origin", origin: "https://example.com", configured: []string{"https://example.com"}, want: "https://example.com"},
		{name: "wildcard config", origin: "https://anything.io", configured: []string{"*"}, want: "*"},
		{name: "bare host config matches scheme", origin: "http://192.168.1.10:8080", configured: []string{"192.168.1.10:8080"}, want: "http://192.168.1.10:8080"},
		{name: "loopback still allowed with strict config", origin: "http://localhost:5173", configured: []string{"https://example.com"}, want: "http://localhost:5173"},
		{name: "lan blocked when explicit config set", origin: "http://10.0.0.5:9000", configured: []string{"https://example.com"}, want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := corsAllow(tc.origin, tc.configured); got != tc.want {
				t.Fatalf("corsAllow(%q, %v) = %q, want %q", tc.origin, tc.configured, got, tc.want)
			}
		})
	}
}
