package service

import (
	"context"
	"testing"

	"github.com/fedorovmatvey/involta-test/internal/model"
	"github.com/stretchr/testify/assert"
)

type MockStorage struct{}

func (m *MockStorage) Create(ctx context.Context, doc *model.Document) error { return nil }
func (m *MockStorage) GetByID(ctx context.Context, id string) (*model.Document, error) {
	return nil, nil
}
func (m *MockStorage) Update(ctx context.Context, doc *model.Document) error { return nil }
func (m *MockStorage) Delete(ctx context.Context, id string) error           { return nil }
func (m *MockStorage) CheckConnection(ctx context.Context) error             { return nil }

func (m *MockStorage) List(ctx context.Context, params model.PaginationParams) ([]model.Document, int, error) {
	docs := []model.Document{
		{
			ID: "doc-1",
			Items: []model.FirstLevelItem{
				{ID: "item-1", Sort: 10},
				{ID: "item-2", Sort: 50},
				{ID: "item-3", Sort: 5},
			},
		},
		{
			ID: "doc-2",
			Items: []model.FirstLevelItem{
				{ID: "item-A", Sort: 1},
				{ID: "item-B", Sort: 99},
			},
		},
	}
	return docs, 2, nil
}

type MockCache struct{}

func (m *MockCache) Get(id string) (*model.Document, bool) { return nil, false }
func (m *MockCache) Set(id string, doc *model.Document)    {}
func (m *MockCache) Delete(id string)                      {}

func TestService_List_ConcurrencyAndSort(t *testing.T) {
	srv := New(&MockStorage{}, &MockCache{})
	ctx := context.Background()

	list, err := srv.List(ctx, model.PaginationParams{Page: 1, PerPage: 10})

	assert.NoError(t, err)
	assert.NotNil(t, list)
	assert.Equal(t, 2, len(list.Documents))

	assert.Equal(t, "doc-1", list.Documents[0].ID)
	assert.Equal(t, "doc-2", list.Documents[1].ID)

	assert.Equal(t, 50, list.Documents[0].Items[0].Sort)
	assert.Equal(t, 10, list.Documents[0].Items[1].Sort)
	assert.Equal(t, 5, list.Documents[0].Items[2].Sort)

	assert.Equal(t, 99, list.Documents[1].Items[0].Sort)
	assert.Equal(t, 1, list.Documents[1].Items[1].Sort)
}
