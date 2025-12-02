package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/fedorovmatvey/involta-test/internal/model"
	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

const desc = true

type Storage struct {
	db        *reindexer.Reindexer
	namespace string
}

func New(dsn, namespace string) (*Storage, error) {
	db := reindexer.NewReindex(dsn, reindexer.WithCreateDBIfMissing())

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	status := db.Status()
	if status.Err != nil {
		return nil, fmt.Errorf("failed to check reindexer status with dsn %q: %w", dsn, status.Err)
	}

	err := db.OpenNamespace(namespace, reindexer.DefaultNamespaceOptions(), model.Document{})
	if err != nil {
		return nil, fmt.Errorf("failed to open namespace %q: %w", namespace, err)
	}

	storage := &Storage{
		db:        db,
		namespace: namespace,
	}

	log.Printf("Successfully connected to Reindexer, namespace: %s", namespace)

	return storage, nil
}

func (s *Storage) initIndexes() error {
	return nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

func (s *Storage) Create(ctx context.Context, doc *model.Document) error {
	if res, err := s.db.Insert(s.namespace, doc); err != nil && res == 0 {
		return fmt.Errorf("failed to insert document: %w", err)
	}
	return nil
}

func (s *Storage) GetByID(ctx context.Context, id string) (*model.Document, error) {
	query := s.db.Query(s.namespace).
		SetContext(ctx).
		Where("id", reindexer.EQ, id).
		Limit(1)

	it := query.Exec()
	defer it.Close()

	if !it.Next() {
		return nil, fmt.Errorf("document not found")
	}

	doc := it.Object().(*model.Document)
	return doc, nil
}

func (s *Storage) Update(ctx context.Context, doc *model.Document) error {
	if res, err := s.db.Update(s.namespace, doc); err != nil && res == 0 {
		return fmt.Errorf("failed to update document: %w", err)
	}
	return nil
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	query := s.db.Query(s.namespace).
		SetContext(ctx).
		Where("id", reindexer.EQ, id)

	if _, err := query.Delete(); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

func (s *Storage) List(ctx context.Context, params model.PaginationParams) ([]model.Document, int, error) {
	query := s.db.Query(s.namespace).
		SetContext(ctx).
		Sort("created_at", desc).
		Limit(params.PerPage).
		Offset(params.GetOffset()).
		ReqTotal()

	it := query.Exec()

	if err := it.Error(); err != nil {
		return nil, 0, fmt.Errorf("failed query Reindexer: %w", err)
	}
	defer it.Close()

	totalCount := it.TotalCount()

	var documents []model.Document

	for it.Next() {
		doc, ok := it.Object().(*model.Document)
		if !ok {
			return nil, 0, fmt.Errorf("unexpected type %T", it.Object())
		}
		documents = append(documents, *doc)
	}

	if it.Error() != nil {
		return nil, 0, fmt.Errorf("failed while iterating document: %w", it.Error())
	}

	return documents, totalCount, nil
}

func (s *Storage) CheckConnection(ctx context.Context) error {
	query := s.db.Query(s.namespace).SetContext(ctx).
		Limit(1)

	it := query.Exec()
	defer it.Close()

	if it.Error() != nil {
		return fmt.Errorf("failed to query namespace %s: %w", s.namespace, it.Error())
	}

	log.Printf("Namespace '%s' is accessible and ready", s.namespace)
	return nil
}
