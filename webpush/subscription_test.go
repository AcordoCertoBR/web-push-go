package webpush

import "testing"

func TestHasValidEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		valid    bool
	}{
		{name: "https endpoint", endpoint: "https://push.example.com/send/abc", valid: true},
		{name: "http endpoint (local testing)", endpoint: "http://localhost:8080/push", valid: true},
		{name: "empty", endpoint: "", valid: false},
		{name: "opaque string", endpoint: "not-a-url", valid: false},
		{name: "missing host", endpoint: "https://", valid: false},
		{name: "unsupported scheme", endpoint: "ftp://push.example.com/send", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Subscription{Endpoint: tt.endpoint}
			if got := s.HasValidEndpoint(); got != tt.valid {
				t.Fatalf("HasValidEndpoint(%q) = %v, want %v", tt.endpoint, got, tt.valid)
			}
		})
	}
}
