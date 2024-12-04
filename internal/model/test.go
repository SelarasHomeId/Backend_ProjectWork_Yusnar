package model

import "selarashomeid/internal/abstraction"

type TestEntity struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
	IsDelete bool   `json:"is_delete"`
}

// TestEntityModel ...
type TestEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	TestEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (TestEntityModel) TableName() string {
	return "test"
}

type BannerCountDataModel struct {
	Count int `json:"count"`
}
