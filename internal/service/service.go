package service

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/fedorovmatvey/involta-test/internal/model"
	"github.com/google/uuid"
)

type documentStorage interface {
	Create(ctx context.Context, doc *model.Document) error
	GetByID(ctx context.Context, id string) (*model.Document, error)
	Update(ctx context.Context, doc *model.Document) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params model.PaginationParams) ([]model.Document, int, error)
	CheckConnection(ctx context.Context) error
}

type documentCache interface {
	Get(id string) (*model.Document, bool)
	Set(id string, doc *model.Document)
	Delete(id string)
}
type Service struct {
	storage documentStorage
	cache   documentCache
}

func New(storage documentStorage, cache documentCache) *Service {
	return &Service{
		storage: storage,
		cache:   cache,
	}
}

func (s *Service) Create(ctx context.Context, req model.CreateDocumentRequest) (*model.Document, error) {
	doc := &model.Document{
		ID:          generateID(),
		Title:       req.Title,
		Description: req.Description,
		Items:       req.Items,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.storage.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	return doc, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*model.Document, error) {
	if cachedDoc, found := s.cache.Get(id); found {
		processedDoc := s.processDocument(cachedDoc)
		return processedDoc, nil
	}

	doc, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	s.cache.Set(id, doc)

	processedDoc := s.processDocument(doc)
	return processedDoc, nil
}

func (s *Service) Update(ctx context.Context, id string, req model.UpdateDocumentRequest) (*model.Document, error) {
	doc, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	if req.Title != nil {
		doc.Title = *req.Title
	}
	if req.Description != nil {
		doc.Description = *req.Description
	}
	if req.Items != nil {
		doc.Items = *req.Items
	}
	doc.UpdatedAt = time.Now()

	if err := s.storage.Update(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	s.cache.Delete(id)

	return doc, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.storage.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	s.cache.Delete(id)

	return nil
}

func (s *Service) List(ctx context.Context, params model.PaginationParams) (*model.DocumentList, error) {
	params.Validate()

	documents, total, err := s.storage.List(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	processedDocs, err := s.processDocumentsParallel(ctx, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to process documents: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

	return &model.DocumentList{
		Documents:  processedDocs,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) processDocument(doc *model.Document) *model.Document {
	processed := *doc

	if doc.Items != nil {
		processed.Items = make([]model.FirstLevelItem, len(doc.Items))
		copy(processed.Items, doc.Items)
	}

	sort.Slice(processed.Items, func(i, j int) bool {
		return processed.Items[i].Sort > processed.Items[j].Sort
	})

	return &processed
}
func (s *Service) processDocumentsParallel(ctx context.Context, documents []model.Document) ([]model.Document, error) {
	if len(documents) == 0 {
		return documents, nil
	}

	sem := make(chan struct{}, runtime.NumCPU())

	type result struct {
		index int
		doc   *model.Document
	}

	results := make(chan result, len(documents))
	var wg sync.WaitGroup

	for i, doc := range documents {
		if ctx.Err() != nil {
			break
		}

		select {
		case sem <- struct{}{}:

		case <-ctx.Done():
			break
		}

		wg.Add(1)
		go func(idx int, d model.Document) {
			defer wg.Done()

			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return
			default:
				processed := s.processDocument(&d)
				results <- result{index: idx, doc: processed}
			}
		}(i, doc)
	}

	wg.Wait()
	close(results)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	processedMap := make(map[int]*model.Document)
	for r := range results {
		processedMap[r.index] = r.doc
	}

	if len(processedMap) != len(documents) {
		return nil, fmt.Errorf("processing incomplete: expected %d documents, got %d", len(documents), len(processedMap))
	}

	processed := make([]model.Document, len(documents))
	for i := 0; i < len(documents); i++ {
		processed[i] = *processedMap[i]
	}

	return processed, nil
}

func generateID() string {
	return uuid.NewString()
}
