package model

import (
	"selarashomeid/internal/abstraction"
	"time"
)

type AffiliateEntity struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Instagram string    `json:"instagram"`
	Tiktok    string    `json:"tiktok"`
	CreatedAt time.Time `json:"created_at"`
}

// AffiliateEntityModel ...
type AffiliateEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	AffiliateEntity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (AffiliateEntityModel) TableName() string {
	return "affiliate"
}
