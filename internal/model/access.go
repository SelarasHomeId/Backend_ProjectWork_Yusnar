package model

import (
	"selarashomeid/internal/abstraction"
	"time"
)

type AccessEntity struct {
	Module    string    `json:"module"`
	CreatedAt time.Time `json:"created_at"`
}

// AccessEntityModel ...
type AccessEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	AccessEntity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (AccessEntityModel) TableName() string {
	return "access"
}
