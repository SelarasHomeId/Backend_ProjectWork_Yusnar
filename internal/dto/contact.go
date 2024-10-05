package dto

import (
	"selarashomeid/internal/abstraction"

	"gorm.io/gorm"
)

type ContactCreateRequest struct {
	Name    string `json:"name" query:"name"`
	Email   string `json:"email" query:"email"`
	Phone   string `json:"phone" query:"phone"`
	Message string `json:"message" query:"message"`
}

type ContactFilter struct {
	Search *string `json:"search" query:"search"`
}

func (f ContactFilter) Apply(tx *gorm.DB, ctx *abstraction.Context) *gorm.DB {
	if f.Search != nil {
		search := "%" + *f.Search + "%"
		tx.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", search, search, search)
	}
	return tx
}

type ContactFindByIDRequest struct {
	ID int `json:"id" param:"id" query:"id"`
}

type ContactDeleteByIDRequest struct {
	ID int `json:"id" param:"id" query:"id"`
}

type ContactGetCountResponse struct {
	CountContact int `json:"count_contact"`
}
