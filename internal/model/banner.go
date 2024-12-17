package model

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type BannerEntity struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
	IsDelete bool   `json:"is_delete"`
	IsPopup  bool   `json:"is_popup"`
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

func (m *BannerEntityModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedAt = general.NowLocal()
	return
}

func (m *BannerEntityModel) BeforeCreate(tx *gorm.DB) (err error) {
	m.CreatedAt = *general.NowLocal()
	return
}
