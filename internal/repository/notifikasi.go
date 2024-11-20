package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Notifikasi interface {
	Create(ctx *abstraction.Context, data *model.NotifikasiEntityModel) *gorm.DB
	FindByUserId(ctx *abstraction.Context, userId *int) (data []*model.NotifikasiEntityModel, err error)
	CountByUserId(ctx *abstraction.Context, userId *int) (countTotal *int, countRead *int, countUnread *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.NotifikasiEntityModel, error)
	Update(ctx *abstraction.Context, data *model.NotifikasiEntityModel) *gorm.DB
}

type notifikasi struct {
	abstraction.Repository
}

func NewNotifikasi(db *gorm.DB) *notifikasi {
	return &notifikasi{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *notifikasi) Create(ctx *abstraction.Context, data *model.NotifikasiEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *notifikasi) FindByUserId(ctx *abstraction.Context, userId *int) (data []*model.NotifikasiEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "notifikasi", fmt.Sprintf("user_id = %d", *userId))
	order := general.ProcessOrder(ctx)
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Find(&data).
		Error
	return
}

func (r *notifikasi) CountByUserId(ctx *abstraction.Context, userId *int) (countTotal *int, countRead *int, countUnread *int, err error) {
	var count model.NotifikasiCountDataModel
	whereTotal, whereParamTotal := general.ProcessWhereParam(ctx, "notifikasi", fmt.Sprintf("user_id = %d", *userId))
	err = r.CheckTrx(ctx).
		Table("notifikasi").
		Select(`
			COUNT(*) AS count_total, 
			COUNT(CASE WHEN is_read = TRUE THEN 1 END) AS count_read,
			COUNT(CASE WHEN is_read = FALSE THEN 1 END) AS count_unread
		`).
		Where(whereTotal, whereParamTotal).
		Find(&count).
		Error
	countTotal = &count.CountTotal
	countRead = &count.CountRead
	countUnread = &count.CountUnread
	return
}

func (r *notifikasi) FindById(ctx *abstraction.Context, id int) (*model.NotifikasiEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.NotifikasiEntityModel
	err := conn.
		Where("id = ?", id).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *notifikasi) Update(ctx *abstraction.Context, data *model.NotifikasiEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}
