package models

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const maxTitleLength = 255

type Task struct {
	ID          int       `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Completed   bool      `json:"completed" db:"completed"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type UpdateTaskInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Completed   *bool   `json:"completed"`
}

func (in *CreateTaskInput) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	return validateTitle(in.Title)
}

func (in *UpdateTaskInput) Validate() error {
	if in.Title == nil && in.Description == nil && in.Completed == nil {
		return errors.New("no fields to update")
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		in.Title = &title
		if err := validateTitle(title); err != nil {
			return err
		}
	}
	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		in.Description = &description
	}
	return nil
}

func validateTitle(title string) error {
	if title == "" {
		return errors.New("title is required")
	}
	if utf8.RuneCountInString(title) > maxTitleLength {
		return errors.New("title must be at most 255 characters")
	}
	return nil
}
