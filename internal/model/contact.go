package model

import (
	"selarashomeid/internal/abstraction"

	"gorm.io/gorm"
)

type ContactEntity struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// ContactEntityModel ...
type ContactEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	ContactEntity

	abstraction.EntityJustCreated

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (ContactEntityModel) TableName() string {
	return "contact"
}

type ContactCountDataModel struct {
	Count int `json:"count"`
}

func (m *ContactEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
