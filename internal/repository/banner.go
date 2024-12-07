package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Banner interface {
	Find(ctx *abstraction.Context) (data []*model.BannerEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
}

type banner struct {
	abstraction.Repository
}

func NewBanner(db *gorm.DB) *banner {
	return &banner{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *banner) Find(ctx *abstraction.Context) (data []*model.BannerEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "banner", "is_delete = @false")
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

func (r *banner) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "banner", "is_delete = @false")
	var count model.BannerCountDataModel
	err = r.CheckTrx(ctx).
		Table("banner").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}
