package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskCommentEntity struct {
	TaskId    int    `json:"task_id"`
	Comment   string `json:"comment"`
	IsDelete  bool   `json:"is_delete"`
	IsHistory bool   `json:"is_history"`
}

// TaskCommentEntityModel ...
type TaskCommentEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskCommentEntity

	abstraction.EntityWithBy

	CreateBy UserEntityModel `json:"create_by" gorm:"foreignKey:CreatedBy"`
	UpdateBy UserEntityModel `json:"update_by" gorm:"foreignKey:UpdatedBy"`

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
	m.UpdatedBy = &m.Context.Auth.ID
	return
}

func (m *TaskCommentEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedBy = m.Context.Auth.ID
	if m.IsHistory {
		m.CreatedBy = constant.USER_ID_SYSTEM
	}
	// m.CreatedAt = *general.Now()
	return
}
