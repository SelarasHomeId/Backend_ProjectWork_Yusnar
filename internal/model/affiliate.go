package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type AffiliateEntity struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Instagram string `json:"instagram"`
	Tiktok    string `json:"tiktok"`
	Info      string `json:"info"`
}

// AffiliateEntityModel ...
type AffiliateEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	AffiliateEntity

	abstraction.EntityJustCreated

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (AffiliateEntityModel) TableName() string {
	return "affiliate"
}

type AffiliateCountDataModel struct {
	Count int `json:"count"`
}

func (m *AffiliateEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.Now()
	return
}
