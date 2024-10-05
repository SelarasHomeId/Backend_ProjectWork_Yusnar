package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type Notification interface {
	Create(ctx *abstraction.Context, data *model.NotificationEntityModel) *gorm.DB
	GetAll(ctx *abstraction.Context) ([]*model.NotificationEntityModel, error)
	CountUnread(ctx *abstraction.Context) (*int, error)
	SetRead(ctx *abstraction.Context, data *model.NotificationEntityModel) *gorm.DB
	FindByID(ctx *abstraction.Context, id int) (data *model.NotificationEntityModel, err error)
	DeleteByDataID(ctx *abstraction.Context, dataId int) *gorm.DB
}

type notification struct {
	abstraction.Repository
}

func NewNotification(db *gorm.DB) *notification {
	return &notification{
		abstraction.Repository{
			Db: db,
		},
	}
}

func (r *notification) Create(ctx *abstraction.Context, data *model.NotificationEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *notification) GetAll(ctx *abstraction.Context) (data []*model.NotificationEntityModel, err error) {
	err = r.CheckTrx(ctx).Order("id DESC").Find(&data).Error
	return
}

func (r *notification) CountUnread(ctx *abstraction.Context) (*int, error) {
	data := &model.CountNotificationUnread{}
	err := r.CheckTrx(ctx).Model(&model.NotificationEntityModel{}).Select("COUNT(*) AS count_unread").Where("is_read = false").Find(&data.CountUnread).Error
	return &data.CountUnread, err
}

func (r *notification) SetRead(ctx *abstraction.Context, data *model.NotificationEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update("is_read", true)
}

func (r *notification) FindByID(ctx *abstraction.Context, id int) (data *model.NotificationEntityModel, err error) {
	err = r.CheckTrx(ctx).Where("id = ?", id).Take(&data).Error
	return
}

func (r *notification) DeleteByDataID(ctx *abstraction.Context, dataId int) *gorm.DB {
	return r.CheckTrx(ctx).Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("data_id = ?", dataId)
	}).Delete(&model.NotificationEntityModel{})
}
