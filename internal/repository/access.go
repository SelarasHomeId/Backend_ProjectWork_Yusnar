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
	GetCount(ctx *abstraction.Context) (data dto.AccessGetCountResponse, err error)
}

type access struct {
	abstraction.Repository
}

func NewAccess(db *gorm.DB) *access {
	return &access{
		abstraction.Repository{
			Db: db,
		},
	}
}

func (r *access) Create(ctx *abstraction.Context, data *model.AccessEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *access) GetCount(ctx *abstraction.Context) (data dto.AccessGetCountResponse, err error) {
	currentMonth := int(general.NowLocal().Month())
	currentYear := general.NowLocal().Year()

	err = r.CheckTrx(ctx).Raw(`
		SELECT
			SUM(CASE WHEN module = 'instagram' THEN 1 ELSE 0 END) as count_instagram,
			SUM(CASE WHEN module = 'tiktok' THEN 1 ELSE 0 END) as count_tiktok,
			SUM(CASE WHEN module = 'facebook' THEN 1 ELSE 0 END) as count_facebook,
			SUM(CASE WHEN module = 'whatsapp' THEN 1 ELSE 0 END) as count_whatsapp
		FROM access
		WHERE MONTH(created_at) = ? AND YEAR(created_at) = ?
	`, currentMonth, currentYear).Scan(&data).Error

	if err != nil {
		return data, err
	}

	return data, nil
}
