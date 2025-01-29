package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"
	"time"

	"gorm.io/gorm"
)

type ChecklistItemEntity struct {
	Title           string     `json:"title"`
	TaskChecklistId int        `json:"task_checklist_id"`
	AssignToUser    *string    `json:"assign_to_user"`
	DueDate         *time.Time `json:"due_date"`
	IsDelete        bool       `json:"is_delete"`
	IsCompleted     bool       `json:"is_completed"`
	SortNumber      int        `json:"sort_number"`
}

// ChecklistItemEntityModel ...
type ChecklistItemEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	ChecklistItemEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (ChecklistItemEntityModel) TableName() string {
	return "checklist_item"
}

type ChecklistItemCountDataModel struct {
	Count int `json:"count"`
}

func (m *ChecklistItemEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *ChecklistItemEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	// m.CreatedAt = *general.NowLocal()
	return
}
