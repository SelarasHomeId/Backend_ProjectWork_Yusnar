package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Contact interface {
	Create(ctx *abstraction.Context, data *model.ContactEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.ContactEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
}

type contact struct {
	abstraction.Repository
}

func NewContact(db *gorm.DB) *contact {
	return &contact{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *contact) Create(ctx *abstraction.Context, data *model.ContactEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *contact) Find(ctx *abstraction.Context) (data []*model.ContactEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "contact", "")
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

func (r *contact) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "contact", "")
	var count model.ContactCountDataModel
	err = r.CheckTrx(ctx).
		Table("contact").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}
