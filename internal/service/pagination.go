package service

import "strconv"

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Pagination struct {
	Page     int
	PageSize int
}

func ParsePagination(pageStr, pageSizeStr string) (Pagination, error) {
	page := DefaultPage
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return Pagination{}, ErrInvalidPagination
		}
		page = p
	}

	pageSize := DefaultPageSize
	if pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 || ps > MaxPageSize {
			return Pagination{}, ErrInvalidPagination
		}
		pageSize = ps
	}

	return Pagination{Page: page, PageSize: pageSize}, nil
}
