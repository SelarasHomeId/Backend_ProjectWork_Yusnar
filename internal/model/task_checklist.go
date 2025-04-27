package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskChecklistEntity struct {
	Title    string `json:"title"`
	TaskId   int    `json:"task_id"`
	IsDelete bool   `json:"is_delete"`
}

// TaskChecklistEntityModel ...
type TaskChecklistEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskChecklistEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TaskChecklistEntityModel) TableName() string {
	return "task_checklist"
}

type TaskChecklistCountDataModel struct {
	Count int `json:"count"`
}

func (m *TaskChecklistEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *TaskChecklistEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.Now()
	return
}
