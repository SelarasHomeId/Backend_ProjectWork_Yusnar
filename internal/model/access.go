package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type AccessEntity struct {
	Module string `json:"module"`
}

// AccessEntityModel ...
type AccessEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	AccessEntity

	abstraction.EntityJustCreated

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (AccessEntityModel) TableName() string {
	return "access"
}

func (m *AccessEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.Now()
	return
}
