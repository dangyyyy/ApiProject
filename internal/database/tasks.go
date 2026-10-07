package database

import (
	"apiproject/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const taskColumns = `id, title, description, completed, created_at, updated_at`

type TaskStore struct {
	db *sqlx.DB
}

func NewTaskStore(db *sqlx.DB) *TaskStore {
	return &TaskStore{db: db}
}

func (s *TaskStore) GetAll(ctx context.Context) ([]models.Task, error) {
	tasks := []models.Task{}
	query := `SELECT ` + taskColumns + ` FROM tasks ORDER BY created_at DESC, id DESC`
	if err := s.db.SelectContext(ctx, &tasks, query); err != nil {
		return nil, fmt.Errorf("select all tasks: %w", err)
	}
	return tasks, nil
}

func (s *TaskStore) GetByID(ctx context.Context, id int) (*models.Task, error) {
	var task models.Task
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`
	err := s.db.GetContext(ctx, &task, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task %d: %w", id, err)
	}
	return &task, nil
}

func (s *TaskStore) Create(ctx context.Context, input *models.CreateTaskInput) (*models.Task, error) {
	var task models.Task
	query := `INSERT INTO tasks (title, description, completed)
VALUES ($1, $2, $3)
RETURNING ` + taskColumns
	err := s.db.GetContext(ctx, &task, query, input.Title, input.Description, input.Completed)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return &task, nil
}

func (s *TaskStore) Update(ctx context.Context, id int, input *models.UpdateTaskInput) (*models.Task, error) {
	var task models.Task
	query := `UPDATE tasks
SET title       = COALESCE($1, title),
    description = COALESCE($2, description),
    completed   = COALESCE($3, completed),
    updated_at  = NOW()
WHERE id = $4
RETURNING ` + taskColumns
	err := s.db.GetContext(ctx, &task, query, input.Title, input.Description, input.Completed, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update task %d: %w", id, err)
	}
	return &task, nil
}

func (s *TaskStore) Delete(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task %d: rows affected: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
