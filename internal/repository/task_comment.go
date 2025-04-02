package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskComment interface {
	FindByTaskId(ctx *abstraction.Context, task_id int) (data []*model.TaskCommentEntityModel, err error)
	CountByTaskId(ctx *abstraction.Context, task_id int) (data *int, err error)
	Create(ctx *abstraction.Context, data *model.TaskCommentEntityModel) *gorm.DB
	FindById(ctx *abstraction.Context, id int) (*model.TaskCommentEntityModel, error)
	Update(ctx *abstraction.Context, data *model.TaskCommentEntityModel) *gorm.DB
}

type task_comment struct {
	abstraction.Repository
}

func NewTaskComment(db *gorm.DB) *task_comment {
	return &task_comment{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *task_comment) FindByTaskId(ctx *abstraction.Context, task_id int) (data []*model.TaskCommentEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_comment", "is_delete = @false"+fmt.Sprintf(" AND task_id = %d", task_id))
	limit, offset := general.ProcessLimitOffset(ctx)
	order := "created_at DESC"
	err = r.CheckTrx(ctx).
		Where(where, whereParam).
		Order(order).
		Limit(limit).
		Offset(offset).
		Preload("CreateBy").
		Preload("UpdateBy").
		Find(&data).
		Error
	return
}

func (r *task_comment) CountByTaskId(ctx *abstraction.Context, task_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_comment", "is_delete = @false"+fmt.Sprintf(" AND task_id = %d", task_id))
	var count model.TaskCommentCountDataModel
	err = r.CheckTrx(ctx).
		Table("task_comment").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task_comment) Create(ctx *abstraction.Context, data *model.TaskCommentEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *task_comment) FindById(ctx *abstraction.Context, id int) (*model.TaskCommentEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TaskCommentEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *task_comment) Update(ctx *abstraction.Context, data *model.TaskCommentEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}
