package model

import "selarashomeid/internal/abstraction"

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
