package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type NotifikasiEntity struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	IsRead  bool   `json:"is_read"`
	UserId  int    `json:"user_id"`
	Link    string `json:"link"`
}

// NotifikasiEntityModel ...
type NotifikasiEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	NotifikasiEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (NotifikasiEntityModel) TableName() string {
	return "notifikasi"
}

type NotifikasiCountDataModel struct {
	CountTotal  int `json:"count_total"`
	CountRead   int `json:"count_read"`
	CountUnread int `json:"count_unread"`
}

func (m *NotifikasiEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *NotifikasiEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.NowLocal()
	return
}
