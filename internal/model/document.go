package model

import "time"

type Document struct {
	ID          string           `json:"id" reindex:"id,,pk"`
	Title       string           `json:"title" reindex:"title"`
	Description string           `json:"description" reindex:"description"`
	CreatedAt   time.Time        `json:"created_at" reindex:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" reindex:"updated_at"`
	Items       []FirstLevelItem `json:"items" reindex:"items"`
	Internal    string           `reindex:"internal"`
}

type FirstLevelItem struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Sort        int               `json:"sort"`
	Value       string            `json:"value"`
	SecondLevel []SecondLevelItem `json:"second_level"`
	MetaData    string            `json:"-"`
}

type SecondLevelItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	PrivateInfo string `json:"-"`
}

type DocumentList struct {
	Documents  []Document `json:"documents"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
	TotalPages int        `json:"total_pages"`
}

type CreateDocumentRequest struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Items       []FirstLevelItem `json:"items"`
}

type UpdateDocumentRequest struct {
	Title       *string           `json:"title,omitempty"`
	Description *string           `json:"description,omitempty"`
	Items       *[]FirstLevelItem `json:"items,omitempty"`
}

type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func (p *PaginationParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 10
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
}

func (p *PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.PerPage
}
