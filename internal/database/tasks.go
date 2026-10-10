package database

import (
	"apiproject/internal/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

const taskColumns = `id, title, description, completed, created_at, updated_at`

type TaskStore struct {
	db *sqlx.DB
}

func NewTaskStore(db *sqlx.DB) *TaskStore {
	return &TaskStore{db: db}
}

var sortColumns = map[string]string{
	"created_at": "created_at",
	"title":      "title",
}

func (s *TaskStore) List(ctx context.Context, f models.TaskFilter) ([]models.Task, int, error) {
	var conditions []string
	var args []any

	addCondition := func(condition string, arg any) {
		args = append(args, arg)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}

	if f.Completed != nil {
		addCondition("completed = $%d", *f.Completed)
	}
	if f.Search != "" {
		addCondition("title ILIKE $%d", "%"+escapeLike(f.Search)+"%")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM tasks`+where, args...); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	column, ok := sortColumns[f.Sort]
	if !ok {
		column = "created_at"
	}
	direction := "DESC"
	if f.Order == "asc" {
		direction = "ASC"
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(`SELECT %s FROM tasks%s ORDER BY %s %s, id %s LIMIT $%d OFFSET $%d`,
		taskColumns, where, column, direction, direction, len(args)-1, len(args))

	tasks := []models.Task{}
	if err := s.db.SelectContext(ctx, &tasks, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, total, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
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
