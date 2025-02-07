package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type WorkspaceEntity struct {
	ProjectId int    `json:"project_id"`
	Name      string `json:"name"`
	IsDelete  bool   `json:"is_delete"`
}

// WorkspaceEntityModel ...
type WorkspaceEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	WorkspaceEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (WorkspaceEntityModel) TableName() string {
	return "workspace"
}

type WorkspaceCountDataModel struct {
	Count int `json:"count"`
}

func (m *WorkspaceEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *WorkspaceEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
