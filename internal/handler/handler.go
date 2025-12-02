package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/fedorovmatvey/involta-test/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	_ "github.com/fedorovmatvey/involta-test/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

type documentService interface {
	Create(ctx context.Context, req model.CreateDocumentRequest) (*model.Document, error)
	GetByID(ctx context.Context, id string) (*model.Document, error)
	Update(ctx context.Context, id string, req model.UpdateDocumentRequest) (*model.Document, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, params model.PaginationParams) (*model.DocumentList, error)
}
type Handler struct {
	service documentService
}

func New(service documentService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) InitRoutes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger) // Встроенный логгер chi очень удобен
	r.Use(middleware.Recoverer)

	r.Get("/health", h.HealthCheck)
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1/documents", func(r chi.Router) {
		r.Get("/", h.ListDocuments)
		r.Post("/", h.CreateDocument)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetDocumentById)
			r.Put("/", h.UpdateDocument)
			r.Delete("/", h.DeleteDocument)
		})
	})

	return r
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// ListDocuments retrieves a paginated list of documents
// @Summary List Documents
// @Description Get all documents with pagination and sorting
// @Tags documents
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(10)
// @Success 200 {object} model.DocumentList
// @Failure 500 {object} map[string]string
// @Router /api/v1/documents [get]
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	params := model.PaginationParams{
		Page:    parseIntQuery(r, "page", 1),
		PerPage: parseIntQuery(r, "per_page", 10),
	}

	list, err := h.service.List(ctx, params)
	if err != nil {
		log.Printf("Failed to list documents: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}

	respondJSON(w, http.StatusOK, list)
}

// CreateDocument creates a new document
// @Summary Create Document
// @Description Create a new document with nested items
// @Tags documents
// @Accept json
// @Produce json
// @Param input body model.CreateDocumentRequest true "Document payload"
// @Success 201 {object} model.Document
// @Failure 400 {object} map[string]string
// @Router /api/v1/documents [post]
func (h *Handler) CreateDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req model.CreateDocumentRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	doc, err := h.service.Create(ctx, req)
	if err != nil {
		log.Printf("Failed to create document: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create document")
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

// GetDocumentById gets a document
// @Summary Get Document
// @Description Get a document by ID (cached)
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} model.Document
// @Failure 404 {object} map[string]string
// @Router /api/v1/documents/{id} [get]
func (h *Handler) GetDocumentById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "document id is required")
		return
	}

	doc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		log.Printf("Failed to get document: %v", err)
		respondError(w, http.StatusNotFound, "document not found")
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// UpdateDocument updates a document
// @Summary Update Document
// @Description Update fields of an existing document
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param input body model.UpdateDocumentRequest true "Update payload"
// @Success 200 {object} model.Document
// @Router /api/v1/documents/{id} [put]
func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "document id is required")
		return
	}

	var req model.UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	doc, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		log.Printf("Failed to update document: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update document")
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// DeleteDocument deletes a document
// @Summary Delete Document
// @Description Remove a document by ID
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/documents/{id} [delete]
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "document id is required")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		log.Printf("Failed to delete document: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "document deleted successfully",
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}

func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
