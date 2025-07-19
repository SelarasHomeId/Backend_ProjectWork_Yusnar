package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Affiliate interface {
	Create(ctx *abstraction.Context, data *model.AffiliateEntityModel) *gorm.DB
	CreateWithoutContext(ctx *abstraction.Context, data *model.AffiliateEntityModel) *gorm.DB
	Find(ctx *abstraction.Context, no_paging bool) (data []*model.AffiliateEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
}

type affiliate struct {
	abstraction.Repository
}

func NewAffiliate(db *gorm.DB) *affiliate {
	return &affiliate{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *affiliate) Create(ctx *abstraction.Context, data *model.AffiliateEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *affiliate) Find(ctx *abstraction.Context, no_paging bool) (data []*model.AffiliateEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "affiliate", "")
	limit, offset := general.ProcessLimitOffset(ctx, no_paging)
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

func (r *affiliate) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "affiliate", "")
	var count model.AffiliateCountDataModel
	err = r.CheckTrx(ctx).
		Table("affiliate").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *affiliate) CreateWithoutContext(ctx *abstraction.Context, data *model.AffiliateEntityModel) *gorm.DB {
	return r.Db.Create(data)
}
