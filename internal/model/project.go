package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type ProjectEntity struct {
	Name      string  `json:"name"`
	Location  *string `json:"location"`
	IsDelete  bool    `json:"is_delete"`
	Cover     *string `json:"cover"`
	CoverName *string `json:"cover_name"`
}

// ProjectEntityModel ...
type ProjectEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	ProjectEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (ProjectEntityModel) TableName() string {
	return "project"
}

type ProjectCountDataModel struct {
	Count int `json:"count"`
}

func (m *ProjectEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *ProjectEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
