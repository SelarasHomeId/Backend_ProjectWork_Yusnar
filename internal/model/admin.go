package model

import "selarashomeid/internal/abstraction"

type AdminEntity struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	IsLogin  bool   `json:"is_login"`
}

// AdminEntityModel ...
type AdminEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	AdminEntity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (AdminEntityModel) TableName() string {
	return "admin"
}
