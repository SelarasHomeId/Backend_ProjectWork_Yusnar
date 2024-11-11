package model

import (
	"selarashomeid/internal/abstraction"
)

type UserEntity struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	RoleId    int    `json:"role_id"`
	DivisiId  int    `json:"divisi_id"`
	IsDelete  bool   `json:"is_delete"`
	IsLocked  bool   `json:"is_locked"`
	LoginFrom string `json:"login_from"`
}

// UserEntityModel ...
type UserEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	UserEntity

	abstraction.Entity

	Role   RoleEntityModel   `json:"role" gorm:"foreignKey:RoleId"`
	Divisi DivisiEntityModel `json:"divisi" gorm:"foreignKey:DivisiId"`

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (UserEntityModel) TableName() string {
	return "user"
}

type UserCountDataModel struct {
	Count int `json:"count"`
}
