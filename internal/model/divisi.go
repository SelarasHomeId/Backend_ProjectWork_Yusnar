package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type DivisiEntity struct {
	Name     string `json:"name"`
	IsDelete bool   `json:"is_delete"`
}

// DivisiEntityModel ...
type DivisiEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	DivisiEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (DivisiEntityModel) TableName() string {
	return "divisi"
}

type DivisiCountDataModel struct {
	Count int `json:"count"`
}

func (m *DivisiEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *DivisiEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.NowLocal()
	return
}
