package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskChecklist interface {
	Create(ctx *abstraction.Context, data *model.TaskChecklistEntityModel) *gorm.DB
	FindByTaskId(ctx *abstraction.Context, task_id int, no_paging bool) (data []*model.TaskChecklistEntityModel, err error)
	CountByTaskId(ctx *abstraction.Context, task_id int) (data *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.TaskChecklistEntityModel, error)
	Update(ctx *abstraction.Context, data *model.TaskChecklistEntityModel) *gorm.DB
	Find(ctx *abstraction.Context, no_paging bool) (data []*model.TaskChecklistEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
}

type task_checklist struct {
	abstraction.Repository
}

func NewTaskChecklist(db *gorm.DB) *task_checklist {
	return &task_checklist{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *task_checklist) Create(ctx *abstraction.Context, data *model.TaskChecklistEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *task_checklist) FindByTaskId(ctx *abstraction.Context, task_id int, no_paging bool) (data []*model.TaskChecklistEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_checklist", "is_delete = @false"+fmt.Sprintf(" AND task_id = %d", task_id))
	limit, offset := general.ProcessLimitOffset(ctx, no_paging)
	order := "created_at DESC"
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&data).
		Error
	return
}

func (r *task_checklist) CountByTaskId(ctx *abstraction.Context, task_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_checklist", "is_delete = @false"+fmt.Sprintf(" AND task_id = %d", task_id))
	var count model.TaskChecklistCountDataModel
	err = r.CheckTrx(ctx).
		Table("task_checklist").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task_checklist) FindById(ctx *abstraction.Context, id int) (*model.TaskChecklistEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TaskChecklistEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *task_checklist) Update(ctx *abstraction.Context, data *model.TaskChecklistEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *task_checklist) Find(ctx *abstraction.Context, no_paging bool) (data []*model.TaskChecklistEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_checklist", "is_delete = @false")
	limit, offset := general.ProcessLimitOffset(ctx, no_paging)
	order := "created_at DESC"
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&data).
		Error
	return
}

func (r *task_checklist) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_checklist", "is_delete = @false")
	var count model.TaskChecklistCountDataModel
	err = r.CheckTrx(ctx).
		Table("task_checklist").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}
