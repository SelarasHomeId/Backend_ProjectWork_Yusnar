package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"
	"time"

	"gorm.io/gorm"
)

type TaskEntity struct {
	BoardId      int        `json:"board_id"`
	Title        string     `json:"title"`
	Description  *string    `json:"description"`
	AssignToUser *string    `json:"assign_to_user"`
	Label        *string    `json:"label"`
	IsCompleted  bool       `json:"is_completed"`
	DueDate      *time.Time `json:"due_date"`
	Cover        *string    `json:"cover"`
	CoverName    *string    `json:"cover_name"`
	IsDelete     bool       `json:"is_delete"`
	SortNumber   int        `json:"sort_number"`
}

// TaskEntityModel ...
type TaskEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TaskEntity

	abstraction.EntityWithBy

	CreateBy UserEntityModel `json:"create_by" gorm:"foreignKey:CreatedBy"`
	UpdateBy UserEntityModel `json:"update_by" gorm:"foreignKey:UpdatedBy"`

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TaskEntityModel) TableName() string {
	return "task"
}

type TaskCountDataModel struct {
	Count int `json:"count"`
}

func (m *TaskEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	m.UpdatedBy = &m.Context.Auth.ID
	return
}

func (m *TaskEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.Now()
	m.CreatedBy = m.Context.Auth.ID
	return
}
