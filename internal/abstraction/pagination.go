package abstraction

import (
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"
)

type PaginationInfo struct {
	*Pagination
	Pages       int   `json:"pages"`
	Count       int64 `json:"count"`
	MoreRecords bool  `json:"more_records" example:"false"`
}

// Pagination ...
// swagger:params Pagination
type Pagination struct {
	// Page number of the results page to return.
	// Required: false
	// In: query
	Page *int `query:"page" json:"page" example:"1"`

	// PageSize number of results to return per page.
	// Required: false
	// In: query
	PageSize *int `query:"page_size" json:"page_size" example:"15"`

	// OrderBy field to order the results by.
	// Required: false
	// In: query
	OrderBy *string `query:"order_by" json:"order_by,omitempty" example:"id"`

	// Order direction to order the results by.
	// Required: false
	// In: query
	// Enum: asc, desc
	Order *string `query:"order" json:"order,omitempty" example:"desc"`

	once sync.Once
}

func sanitizeString(input string) string {
	// Menghapus karakter yang bukan huruf, angka, atau underscore
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, input)
}

func (p *Pagination) SetPageInfo(count int64, lenData int) (info *PaginationInfo) {
	if p != nil && p.PageSize != nil {
		info = &PaginationInfo{
			Pagination:  p,
			MoreRecords: false,
			Count:       count,
			Pages:       int(math.Ceil(float64(count) / float64(*p.PageSize))),
		}
		if lenData >= *p.PageSize {
			info.MoreRecords = true
		}
	}
	return
}

func (p *Pagination) SQL() string {
	if p.Page == nil || p.PageSize == nil {
		p.Init()
	}
	return fmt.Sprintf(" LIMIT %d OFFSET %d", *p.PageSize, (*p.Page-1)*(*p.PageSize))
}

func (p *Pagination) SetPage(page int) *Pagination {
	p.Page = &page
	return p
}

func (p *Pagination) SetPageSize(pageSize int) *Pagination {
	p.PageSize = &pageSize
	return p
}

func (p *Pagination) GetOrderBy() string {
	// Validasi dan sanitasi untuk OrderBy
	if p.OrderBy != nil {
		*p.OrderBy = sanitizeString(*p.OrderBy)
		if *p.OrderBy == "" {
			defaultOrderBy := "id"
			p.OrderBy = &defaultOrderBy
		}
	}
	// Validasi dan sanitasi untuk Order
	if p.Order != nil {
		*p.Order = sanitizeString(*p.Order)
		if *p.Order != "asc" && *p.Order != "desc" {
			defaultOrder := "desc"
			p.Order = &defaultOrder
		}
	}
	orderBy := "id"
	if p.OrderBy != nil {
		orderBy = *p.OrderBy
	}
	order := "desc"
	if p.Order != nil {
		order = *p.Order
	}

	return fmt.Sprintf("%s %s", orderBy, order)
}

// NewPagination ...
func NewPagination() *Pagination {
	page := 1
	pageSize := 10
	order := "desc"
	orderBy := "id"

	return &Pagination{
		Page:     &page,
		PageSize: &pageSize,
		Order:    &order,
		OrderBy:  &orderBy,
	}
}

// NewPaginationInfo ...
func (p *Pagination) NewPaginationInfo(data interface{}) (interface{}, *PaginationInfo) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return data, nil
	}

	info := &PaginationInfo{
		Pagination:  p,
		MoreRecords: false,
	}

	if p.PageSize != nil && v.Len() > *p.PageSize {
		data = v.Slice(0, *p.PageSize).Interface()
		info.MoreRecords = true
	}

	return data, info
}

func (p *Pagination) Init() {
	p.once.Do(func() {
		page := 1
		if p.Page != nil && *p.Page > 0 {
			page = *p.Page
		}

		pageSize := 0
		if p.PageSize != nil {
			pageSize = *p.PageSize
		}

		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		p.Page = &page
		p.PageSize = &pageSize
	})
}

func (p *Pagination) GetOffset() int {
	// Validasi dan sanitasi untuk Page
	if p.Page != nil && *p.Page <= 0 {
		defaultPage := 1
		p.Page = &defaultPage
	}
	// Validasi dan sanitasi untuk PageSize
	if p.PageSize != nil && *p.PageSize <= 0 {
		defaultPageSize := 10
		p.PageSize = &defaultPageSize
	} else if p.PageSize != nil && *p.PageSize > 100 {
		maxPageSize := 100
		p.PageSize = &maxPageSize
	}
	return (*p.Page - 1) * *p.PageSize
}

func (p *Pagination) GetLimit() int {
	// Validasi dan sanitasi untuk PageSize
	if p.PageSize != nil && *p.PageSize <= 0 {
		defaultPageSize := 10
		p.PageSize = &defaultPageSize
	} else if p.PageSize != nil && *p.PageSize > 100 {
		maxPageSize := 100
		p.PageSize = &maxPageSize
	}
	return *p.PageSize + 1
}

func (p *Pagination) Apply(tx *gorm.DB) *gorm.DB {
	if p.PageSize != nil {
		p.Init()
		tx.Offset(p.GetOffset()).Limit(p.GetLimit())
	}
	if p.OrderBy != nil {
		tx.Order(p.GetOrderBy())
	}
	return tx
}

func (p *Pagination) Params(params url.Values) {
	if p.Page != nil {
		params.Add("page", strconv.Itoa(*p.Page))
	}
	if p.PageSize != nil {
		params.Add("page_size", strconv.Itoa(*p.PageSize))
	}
	if p.OrderBy != nil {
		params.Add("order_by", *p.OrderBy)
	}
	if p.Order != nil {
		params.Add("order", *p.Order)
	}
}
