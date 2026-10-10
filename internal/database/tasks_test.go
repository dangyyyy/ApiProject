package database

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "go", want: "go"},
		{in: "100%", want: `100\%`},
		{in: "snake_case", want: `snake\_case`},
		{in: `C:\temp`, want: `C:\\temp`},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := escapeLike(tt.in); got != tt.want {
				t.Errorf("escapeLike(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
