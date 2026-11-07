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

func TestApplyPercentageFee(t *testing.T) {
	tests := []struct {
		name          string
		amount        int64
		feePercentage int32
		want          int64
	}{
		{"zero fee", 10000, 0, 10000},
		{"exact ten percent", 10000, 10, 11000},
		{"rounds half up", 15, 10, 17}, // fee = (15*10+50)/100 = 2 -> 17
		{"rounds down", 14, 10, 15},    // fee = (14*10+50)/100 = 1.9 -> 1 -> 15
		{"zero amount", 0, 50, 0},
		{"hundred percent", 100, 100, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyPercentageFee(tt.amount, tt.feePercentage); got != tt.want {
				t.Errorf("ApplyPercentageFee(%d, %d) = %d, want %d", tt.amount, tt.feePercentage, got, tt.want)
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
