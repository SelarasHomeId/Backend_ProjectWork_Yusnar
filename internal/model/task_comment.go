package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskCommentEntity struct {
	TaskId   int    `json:"task_id"`
	Comment  string `json:"comment"`
	IsDelete bool   `json:"is_delete"`
}

// TaskCommentEntityModel ...
type TaskCommentEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskCommentEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TaskCommentEntityModel) TableName() string {
	return "task_comment"
}

type TaskCommentCountDataModel struct {
	Count int `json:"count"`
}

func (m *TaskCommentEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *TaskCommentEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
