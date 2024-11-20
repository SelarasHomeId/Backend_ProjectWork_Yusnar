package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Divisi interface {
	FindById(ctx *abstraction.Context, id int) (*model.DivisiEntityModel, error)
	Create(ctx *abstraction.Context, data *model.DivisiEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.DivisiEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	Update(ctx *abstraction.Context, data *model.DivisiEntityModel) *gorm.DB
}

type divisi struct {
	abstraction.Repository
}

func NewDivisi(db *gorm.DB) *divisi {
	return &divisi{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *divisi) FindById(ctx *abstraction.Context, id int) (*model.DivisiEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.DivisiEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *divisi) Create(ctx *abstraction.Context, data *model.DivisiEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *divisi) Find(ctx *abstraction.Context) (data []*model.DivisiEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "divisi", "is_delete = @false")
	limit, offset := general.ProcessLimitOffset(ctx)
	order := general.ProcessOrder(ctx)
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&data).
		Error
	return
}

func (r *divisi) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "divisi", "is_delete = @false")
	var count model.DivisiCountDataModel
	err = r.CheckTrx(ctx).
		Table("divisi").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *divisi) Update(ctx *abstraction.Context, data *model.DivisiEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}
