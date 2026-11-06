package domain

import (
	"strings"
	"testing"
)

func TestSlugFromName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Café Sol", "cafe-sol"},
		{"  Multiple   Spaces!! ", "multiple-spaces"},
		{"***Leading And Trailing***", "leading-and-trailing"},
		{"Café & Sol 123", "cafe-sol-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SlugFromName(tt.name); got != tt.want {
				t.Errorf("SlugFromName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSlugFromName_AllSymbolsFallsBackToNonEmpty(t *testing.T) {
	got := SlugFromName("!!")
	if got == "" {
		t.Fatal("expected a non-empty fallback slug")
	}
	if !strings.HasPrefix(got, "store-") {
		t.Fatalf("expected fallback slug shaped like \"store-<hex>\", got %q", got)
	}
}
