package webpush

import (
	"testing"
)

func TestFormatVapidSubject(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected string
	}{
		{name: "email without prefix", subject: "dev@example.com", expected: "mailto:dev@example.com"},
		{name: "mailto already formatted", subject: "mailto:dev@example.com", expected: "mailto:dev@example.com"},
		{name: "https", subject: "https://example.com", expected: "https://example.com"},
		{name: "http", subject: "http://example.com", expected: "http://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVapidSubject(tt.subject)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
