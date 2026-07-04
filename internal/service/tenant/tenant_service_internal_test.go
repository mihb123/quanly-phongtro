package tenant

import (
	"testing"
)

func TestRemoveAccents(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Nguyễn Văn A", "nguyenvana"},
		{"Lê Thị Bé", "lethibe"},
		{"", ""},
		{"Hello 123", "hello123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := removeAccents(tt.input)
			if got != tt.want {
				t.Errorf("removeAccents(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
