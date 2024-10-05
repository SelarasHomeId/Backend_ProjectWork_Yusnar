package model

import (
	"selarashomeid/internal/abstraction"
	"time"
)

type ContactEntity struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ContactEntityModel ...
type ContactEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	ContactEntity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (ContactEntityModel) TableName() string {
	return "contact"
}
