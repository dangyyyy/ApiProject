package handlers

import (
	"apiproject/internal/models"
	"context"
)

type fakeStore struct {
	getAllFn  func(ctx context.Context) ([]models.Task, error)
	getByIDFn func(ctx context.Context, id int) (*models.Task, error)
	createFn  func(ctx context.Context, input *models.CreateTaskInput) (*models.Task, error)
	updateFn  func(ctx context.Context, id int, input *models.UpdateTaskInput) (*models.Task, error)
	deleteFn  func(ctx context.Context, id int) error
}

func (f *fakeStore) GetAll(ctx context.Context) ([]models.Task, error) {
	return f.getAllFn(ctx)
}

func (f *fakeStore) GetByID(ctx context.Context, id int) (*models.Task, error) {
	return f.getByIDFn(ctx, id)
}

func (f *fakeStore) Create(ctx context.Context, input *models.CreateTaskInput) (*models.Task, error) {
	return f.createFn(ctx, input)
}

func (f *fakeStore) Update(ctx context.Context, id int, input *models.UpdateTaskInput) (*models.Task, error) {
	return f.updateFn(ctx, id, input)
}

func (f *fakeStore) Delete(ctx context.Context, id int) error {
	return f.deleteFn(ctx, id)
}
