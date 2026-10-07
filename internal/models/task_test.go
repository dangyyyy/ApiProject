package models

import (
	"strings"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

func TestCreateTaskInput_Validate(t *testing.T) {
	tests := []struct {
		name      string
		input     CreateTaskInput
		wantErr   string
		wantTitle string
	}{
		{name: "valid", input: CreateTaskInput{Title: "Buy milk"}, wantTitle: "Buy milk"},
		{name: "trims spaces", input: CreateTaskInput{Title: "  Buy milk  "}, wantTitle: "Buy milk"},
		{name: "empty title", input: CreateTaskInput{Title: ""}, wantErr: "title is required"},
		{name: "only spaces", input: CreateTaskInput{Title: "   "}, wantErr: "title is required"},
		{name: "255 cyrillic runes", input: CreateTaskInput{Title: strings.Repeat("я", 255)}, wantTitle: strings.Repeat("я", 255)},
		{name: "256 runes", input: CreateTaskInput{Title: strings.Repeat("a", 256)}, wantErr: "title must be at most 255 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()

			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.input.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", tt.input.Title, tt.wantTitle)
			}
		})
	}
}

func TestUpdateTaskInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   UpdateTaskInput
		wantErr string
	}{
		{name: "no fields", input: UpdateTaskInput{}, wantErr: "no fields to update"},
		{name: "only completed", input: UpdateTaskInput{Completed: ptr(true)}},
		{name: "valid title", input: UpdateTaskInput{Title: ptr("New title")}},
		{name: "blank title", input: UpdateTaskInput{Title: ptr("   ")}, wantErr: "title is required"},
		{name: "empty description is allowed", input: UpdateTaskInput{Description: ptr("")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()

			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
