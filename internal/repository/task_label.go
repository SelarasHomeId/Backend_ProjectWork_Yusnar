package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskLabel interface {
	Create(ctx *abstraction.Context, data *model.TaskLabelEntityModel) *gorm.DB
	Find(ctx *abstraction.Context, no_paging bool) (data []*model.TaskLabelEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.TaskLabelEntityModel, error)
	Update(ctx *abstraction.Context, data *model.TaskLabelEntityModel) *gorm.DB
}

type task_label struct {
	abstraction.Repository
}

func NewTaskLabel(db *gorm.DB) *task_label {
	return &task_label{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *task_label) Create(ctx *abstraction.Context, data *model.TaskLabelEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *task_label) Find(ctx *abstraction.Context, no_paging bool) (data []*model.TaskLabelEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_label", "is_delete = @false")
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

func (r *task_label) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_label", "is_delete = @false")
	var count model.TaskLabelCountDataModel
	err = r.CheckTrx(ctx).
		Table("task_label").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task_label) FindById(ctx *abstraction.Context, id int) (*model.TaskLabelEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TaskLabelEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *task_label) Update(ctx *abstraction.Context, data *model.TaskLabelEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}
