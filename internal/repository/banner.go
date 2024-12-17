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
	FindById(ctx *abstraction.Context, id int) (*model.BannerEntityModel, error)
	Create(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB
	Update(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB
	GetPopup(ctx *abstraction.Context) (*model.BannerEntityModel, error)
	FindByIdAndPopupTrue(ctx *abstraction.Context, id int) (*model.BannerEntityModel, error)
	FindByPopupTrue(ctx *abstraction.Context) (*model.BannerEntityModel, error)
	UpdateByPopupTrue(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB
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
	where, whereParam := general.ProcessWhereParam(ctx, "banner", "is_delete = @false AND is_popup = @false")
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
	where, whereParam := general.ProcessWhereParam(ctx, "banner", "is_delete = @false AND is_popup = @false")
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

func (r *banner) FindById(ctx *abstraction.Context, id int) (*model.BannerEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BannerEntityModel
	err := conn.
		Where("id = ? AND is_delete = ? AND is_popup = ?", id, false, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *banner) Create(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *banner) Update(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *banner) GetPopup(ctx *abstraction.Context) (*model.BannerEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BannerEntityModel
	err := conn.
		Where("is_delete = ? AND is_popup = ?", false, true).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *banner) FindByIdAndPopupTrue(ctx *abstraction.Context, id int) (*model.BannerEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BannerEntityModel
	err := conn.
		Where("id = ? AND is_delete = ? AND is_popup = ?", id, false, true).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *banner) FindByPopupTrue(ctx *abstraction.Context) (*model.BannerEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BannerEntityModel
	err := conn.
		Where("is_popup = ?", true).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *banner) UpdateByPopupTrue(ctx *abstraction.Context, data *model.BannerEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("is_popup = ?", data.IsPopup).Update("is_delete", data.IsDelete)
}
