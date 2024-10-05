package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Contact interface {
	Create(ctx *abstraction.Context, data *model.ContactEntityModel) *gorm.DB
	Find(ctx *abstraction.Context, f *dto.ContactFilter, p *abstraction.Pagination) ([]*model.ContactEntityModel, *abstraction.PaginationInfo, error)
	FindByID(ctx *abstraction.Context, id int) (data *model.ContactEntityModel, err error)
	DeleteByID(ctx *abstraction.Context, id int) *gorm.DB
	GetAll(ctx *abstraction.Context) ([]*model.ContactEntityModel, error)
	GetCountContact(ctx *abstraction.Context) (data dto.ContactGetCountResponse, err error)
}

type contact struct {
	abstraction.Repository
}

func NewContact(db *gorm.DB) *contact {
	return &contact{
		abstraction.Repository{
			Db: db,
		},
	}
}

func (r *contact) Create(ctx *abstraction.Context, data *model.ContactEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *contact) Find(ctx *abstraction.Context, f *dto.ContactFilter, p *abstraction.Pagination) ([]*model.ContactEntityModel, *abstraction.PaginationInfo, error) {
	var (
		data  []*model.ContactEntityModel
		count int64
		err   error

		info = &abstraction.PaginationInfo{Pagination: p}
	)

	if err = r.CheckTrx(ctx).Model(&model.ContactEntityModel{}).Scopes(func(tx *gorm.DB) *gorm.DB {
		if f != nil {
			f.Apply(tx, ctx)
		}
		return tx
	}).Count(&count).Error; err != nil {
		return nil, nil, err
	}

	if err = r.CheckTrx(ctx).Model(&model.ContactEntityModel{}).Scopes(func(tx *gorm.DB) *gorm.DB {
		if f != nil {
			f.Apply(tx, ctx)
		}
		if p != nil {
			if p.Page == nil || p.PageSize == nil {
				p.Init()
			}
			tx.Offset(p.GetOffset()).Limit(p.GetLimit()).Order(p.GetOrderBy())
		}
		return tx
	}).Find(&data).Error; err != nil {
		return nil, nil, err
	}

	info.Count = count
	return data, info, nil
}

func (r *contact) FindByID(ctx *abstraction.Context, id int) (data *model.ContactEntityModel, err error) {
	err = r.CheckTrx(ctx).Where("id = ?", id).Take(&data).Error
	return
}

func (r *contact) DeleteByID(ctx *abstraction.Context, id int) *gorm.DB {
	return r.CheckTrx(ctx).Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("id = ?", id)
	}).Delete(&model.ContactEntityModel{})
}

func (r *contact) GetAll(ctx *abstraction.Context) (data []*model.ContactEntityModel, err error) {
	err = r.CheckTrx(ctx).Find(&data).Error
	return
}

func (r *contact) GetCountContact(ctx *abstraction.Context) (data dto.ContactGetCountResponse, err error) {
	currentMonth := int(general.NowLocal().Month())
	currentYear := general.NowLocal().Year()

	err = r.CheckTrx(ctx).Raw(`
		SELECT COUNT(*) AS count_contact FROM contact
		WHERE MONTH(created_at) = ? AND YEAR(created_at) = ?
	`, currentMonth, currentYear).Scan(&data).Error

	if err != nil {
		return data, err
	}

	return data, nil
}
