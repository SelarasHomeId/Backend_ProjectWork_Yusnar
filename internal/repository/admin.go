package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type Admin interface {
	FindByUsername(ctx *abstraction.Context, username string) (data *model.AdminEntityModel, err error)
	FindById(ctx *abstraction.Context, id int) (data *model.AdminEntityModel, err error)
	Update(ctx *abstraction.Context, data *model.AdminEntityModel) *gorm.DB
	UpdateLoginTrue(ctx *abstraction.Context, id int) *gorm.DB
	UpdateLoginFalse(ctx *abstraction.Context, id int) *gorm.DB
}

type admin struct {
	abstraction.Repository
}

func NewAdmin(db *gorm.DB) *admin {
	return &admin{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *admin) FindByUsername(ctx *abstraction.Context, username string) (data *model.AdminEntityModel, err error) {
	err = r.CheckTrx(ctx).Where("username = ?", username).Take(&data).Error
	return
}

func (r *admin) FindById(ctx *abstraction.Context, id int) (data *model.AdminEntityModel, err error) {
	err = r.CheckTrx(ctx).Where("id = ?", id).Take(&data).Error
	return
}

func (r *admin) Update(ctx *abstraction.Context, data *model.AdminEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *admin) UpdateLoginTrue(ctx *abstraction.Context, id int) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.AdminEntityModel{}).Where("id = ?", id).Update("is_login", true)
}

func (r *admin) UpdateLoginFalse(ctx *abstraction.Context, id int) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.AdminEntityModel{}).Where("id = ?", id).Update("is_login", false)
}
