package model

import (
	"selarashomeid/internal/abstraction"

	"gorm.io/gorm"
)

type AffiliateEntity struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	Instagram string `json:"instagram"`
	Tiktok    string `json:"tiktok"`
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

func (m *AffiliateEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
