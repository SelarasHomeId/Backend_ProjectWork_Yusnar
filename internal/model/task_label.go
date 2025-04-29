package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskLabelEntity struct {
	Title    string `json:"title"`
	Color    string `json:"color"`
	IsDelete bool   `json:"is_delete"`
}

// TaskLabelEntityModel ...
type TaskLabelEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskLabelEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TaskLabelEntityModel) TableName() string {
	return "task_label"
}

type TaskLabelCountDataModel struct {
	Count int `json:"count"`
}

func (m *TaskLabelEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *TaskLabelEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.Now()
	return
}
