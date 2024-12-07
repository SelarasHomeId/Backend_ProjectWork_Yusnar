package model

import "selarashomeid/internal/abstraction"

type BannerEntity struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
	IsDelete bool   `json:"is_delete"`
}

// BannerEntityModel ...
type BannerEntityModel struct {
	ID int `json:"id" param:"id" form:"id" validate:"number,min=1" gorm:"primaryKey;autoIncrement;"`

	// entity
	BannerEntity

	abstraction.Entity

	// context
	Context *abstraction.Context `json:"-" gorm:"-"`
}

// TableName ...
func (BannerEntityModel) TableName() string {
	return "banner"
}

type BannerCountDataModel struct {
	Count int `json:"count"`
}
