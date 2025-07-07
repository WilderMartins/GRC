package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	DefaultPage = 1
	DefaultPageSize = 10
	MaxPageSize = 100
)

// PaginationQuery holds pagination parameters from the request.
type PaginationQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// PaginatedResponse is a generic struct for paginated API responses.
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	TotalItems int64       `json:"total_items"`
	TotalPages int64       `json:"total_pages"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
}

// GetPaginationParams extracts and validates pagination parameters from Gin context.
func GetPaginationParams(c *gin.Context) (page int, pageSize int) {
	pageQuery := c.DefaultQuery("page", strconv.Itoa(DefaultPage))
	pageSizeQuery := c.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize))

	page, err := strconv.Atoi(pageQuery)
	if err != nil || page < 1 {
		page = DefaultPage
	}

	pageSize, err = strconv.Atoi(pageSizeQuery)
	if err != nil || pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

// PaginateScope returns a GORM scope function to apply pagination.
func PaginateScope(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}
