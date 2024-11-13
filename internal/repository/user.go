package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type User interface {
	FindByEmail(ctx *abstraction.Context, email string) (*model.UserEntityModel, error)
	Create(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.UserEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.UserEntityModel, error)
	Update(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB
	UpdateDelete(ctx *abstraction.Context, id *int, delete bool) *gorm.DB
	UpdateLocked(ctx *abstraction.Context, id *int, locked bool) *gorm.DB
	UpdateLoginFrom(ctx *abstraction.Context, id *int, from string) *gorm.DB
	FindByDivisiId(ctx *abstraction.Context, id *int) (*model.UserEntityModel, error)
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
	err := conn.
		Where("email = ? AND is_delete = ?", email, false).
		Preload("Role").
		Preload("Divisi").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *user) Create(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *user) Find(ctx *abstraction.Context) (data []*model.UserEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "user", "is_delete = @false")
	limit, offset := general.ProcessLimitOffset(ctx)
	order := general.ProcessOrder(ctx)
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Limit(limit).
		Offset(offset).
		Preload("Role").
		Preload("Divisi").
		Find(&data).
		Error
	return
}

func (r *user) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "user", "is_delete = @false ")
	var count model.UserCountDataModel
	err = r.CheckTrx(ctx).
		Table("user").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *user) FindById(ctx *abstraction.Context, id int) (*model.UserEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.UserEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		Preload("Role").
		Preload("Divisi").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *user) Update(ctx *abstraction.Context, data *model.UserEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *user) UpdateDelete(ctx *abstraction.Context, id *int, delete bool) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.UserEntityModel{}).Where("id = ?", id).Update("is_delete", delete)
}

func (r *user) UpdateLocked(ctx *abstraction.Context, id *int, locked bool) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.UserEntityModel{}).Where("id = ?", id).Update("is_locked", locked)
}

func (r *user) UpdateLoginFrom(ctx *abstraction.Context, id *int, from string) *gorm.DB {
	return r.CheckTrx(ctx).Model(&model.UserEntityModel{}).Where("id = ?", id).Update("login_from", from)
}

func (r *user) FindByDivisiId(ctx *abstraction.Context, id *int) (*model.UserEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.UserEntityModel
	err := conn.
		Where("divisi_id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}
