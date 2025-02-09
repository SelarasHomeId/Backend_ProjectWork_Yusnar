package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Project interface {
	FindById(ctx *abstraction.Context, id int) (*model.ProjectEntityModel, error)
	Create(ctx *abstraction.Context, data *model.ProjectEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.ProjectEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	Update(ctx *abstraction.Context, data *model.ProjectEntityModel) *gorm.DB
	UpdateToNull(ctx *abstraction.Context, data *model.ProjectEntityModel, column string) *gorm.DB
}

type project struct {
	abstraction.Repository
}

func NewProject(db *gorm.DB) *project {
	return &project{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *project) FindById(ctx *abstraction.Context, id int) (*model.ProjectEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.ProjectEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *project) Create(ctx *abstraction.Context, data *model.ProjectEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *project) Find(ctx *abstraction.Context) (data []*model.ProjectEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "project", "is_delete = @false")
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

func (r *project) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "project", "is_delete = @false")
	var count model.ProjectCountDataModel
	err = r.CheckTrx(ctx).
		Table("project").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *project) Update(ctx *abstraction.Context, data *model.ProjectEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *project) UpdateToNull(ctx *abstraction.Context, data *model.ProjectEntityModel, column string) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update(column, nil)
}
