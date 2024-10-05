package dto

import (
	"selarashomeid/internal/abstraction"

	"gorm.io/gorm"
)

type AffiliateCreateRequest struct {
	Name      string `json:"name" query:"name"`
	Email     string `json:"email" query:"email"`
	Phone     string `json:"phone" query:"phone"`
	Instagram string `json:"instagram" query:"instagram"`
	Tiktok    string `json:"tiktok" query:"tiktok"`
}

type AffiliateFilter struct {
	Search *string `json:"search" query:"search"`
}

func (f AffiliateFilter) Apply(tx *gorm.DB, ctx *abstraction.Context) *gorm.DB {
	if f.Search != nil {
		search := "%" + *f.Search + "%"
		tx.Where("name LIKE ? OR email LIKE ? OR phone LIKE ?", search, search, search)
	}
	return tx
}

type AffiliateFindByIDRequest struct {
	ID int `json:"id" param:"id" query:"id"`
}

type AffiliateDeleteByIDRequest struct {
	ID int `json:"id" param:"id" query:"id"`
}

type AffiliateGetCountResponse struct {
	CountAffiliate int `json:"count_affiliate"`
}
