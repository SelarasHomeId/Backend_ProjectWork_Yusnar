package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskFileEntity struct {
	TaskId   int    `json:"task_id"`
	File     string `json:"file"`
	IsDelete bool   `json:"is_delete"`
}

// TaskFileEntityModel ...
type TaskFileEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskFileEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TaskFileEntityModel) TableName() string {
	return "task_file"
}

type TaskFileCountDataModel struct {
	Count int `json:"count"`
}

func (m *TaskFileEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *TaskFileEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.NowLocal()
	return
}
