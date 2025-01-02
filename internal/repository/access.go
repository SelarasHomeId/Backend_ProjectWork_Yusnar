package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Access interface {
	Create(ctx *abstraction.Context, data *model.AccessEntityModel) *gorm.DB
	Count(ctx *abstraction.Context) (data *dto.AccesstCountResponse, err error)
}

type access struct {
	abstraction.Repository
}

func NewAccess(db *gorm.DB) *access {
	return &access{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *access) Create(ctx *abstraction.Context, data *model.AccessEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *access) Count(ctx *abstraction.Context) (data *dto.AccesstCountResponse, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "access", "")
	var errCountSosmed error
	var errCountAffiliate error
	var errCountContact error
	errCountSosmed = r.CheckTrx(ctx).
		Table("access").
		Select(`
			SUM(CASE WHEN module = 'instagram' THEN 1 ELSE 0 END) as count_instagram,
	 		SUM(CASE WHEN module = 'tiktok' THEN 1 ELSE 0 END) as count_tiktok,
			SUM(CASE WHEN module = 'facebook' THEN 1 ELSE 0 END) as count_facebook,
			SUM(CASE WHEN module = 'whatsapp' THEN 1 ELSE 0 END) as count_whatsapp
		`).
		Where(where, whereParam).
		Find(&data).
		Error

	errCountAffiliate = r.CheckTrx(ctx).
		Table("affiliate").
		Select(`
			COUNT(*) AS count_affiliate
		`).
		Where(where, whereParam).
		Find(&data).
		Error

	errCountContact = r.CheckTrx(ctx).
		Table("contact").
		Select(`
			COUNT(*) AS count_contact
		`).
		Where(where, whereParam).
		Find(&data).
		Error

	if errCountSosmed != nil {
		err = errCountSosmed
	} else if errCountAffiliate != nil {
		err = errCountAffiliate
	} else {
		err = errCountContact
	}

	return
}
