package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type BoardEntity struct {
	WorkspaceId int    `json:"workspace_id"`
	Name        string `json:"name"`
	TaskTotal   int    `json:"task_total"`
	SortNumber  int    `json:"sort_number"`
	IsDelete    bool   `json:"is_delete"`
}

// BoardEntityModel ...
type BoardEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	BoardEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (BoardEntityModel) TableName() string {
	return "board"
}

type BoardCountDataModel struct {
	Count int `json:"count"`
}

func (m *BoardEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *BoardEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.NowLocal()
	return
}
