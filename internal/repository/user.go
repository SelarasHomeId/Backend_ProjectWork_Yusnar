package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type User interface {
	FindByEmail(ctx *abstraction.Context, email string) (*model.UserEntityModel, error)
	Create(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.UserEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.UserEntityModel, error)
	Update(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB
	UpdateLoginUser(ctx *abstraction.Context, id int, login bool) *gorm.DB
}

type user struct {
	abstraction.Repository
}

func NewUser(db *gorm.DB) *user {
	return &user{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *user) FindByEmail(ctx *abstraction.Context, email string) (*model.UserEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.UserEntityModel
	err := conn.Where("email = ? AND is_delete = ?", email, false).Preload("Role").Preload("Divisi").First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *user) Create(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *user) Find(ctx *abstraction.Context) (data []*model.UserEntityModel, err error) {
	err = r.CheckTrx(ctx).Where("is_delete = ?", false).Preload("Role").Preload("Divisi").Find(&data).Error
	return
}

func (r *user) Count(ctx *abstraction.Context) (data *int, err error) {
	var count model.UserCountDataModel
	err = r.CheckTrx(ctx).Table("user").Select("COUNT(*) AS count").Where("is_delete = ?", false).Find(&count).Error
	data = &count.Count
	return
}

func (r *user) FindById(ctx *abstraction.Context, id int) (*model.UserEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.UserEntityModel
	err := conn.Where("id = ? AND is_delete = ?", id, false).Preload("Role").Preload("Divisi").First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *user) Update(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *user) UpdateLoginUser(ctx *abstraction.Context, id int, login bool) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.UserEntityModel{}).Where("id = ?", id).Update("is_login", login)
}
