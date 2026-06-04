package service

import (
	"testing"
)

func TestGenerateDeterministicSecret(t *testing.T) {
	svc := &zaloServiceImpl{}
	
	tests := []struct {
		input string
		want  string
	}{
		{"short", "short"},
		{"exactly12chr", "exactly12chr"},
		{"thisiswaylongerthan12characters", "thisiswaylon"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := svc.generateDeterministicSecret(tt.input)
			if got != tt.want {
				t.Errorf("generateDeterministicSecret(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b int
		want int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{-1, 0, -1},
		{4, 4, 4},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
