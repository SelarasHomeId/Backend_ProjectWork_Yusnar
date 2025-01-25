package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Task interface {
	Create(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB
	FindById(ctx *abstraction.Context, id int) (*model.TaskEntityModel, error)
	Update(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB
	FindByBoardIdArr(ctx *abstraction.Context, board_id int) (data []*model.TaskEntityModel, err error)
	CountByBoardIdArr(ctx *abstraction.Context, board_id int) (data *int, err error)
	UpdateCompleted(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB
	Find(ctx *abstraction.Context) (data []*model.TaskEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	FindByWorkspace(ctx *abstraction.Context, workspace_id int) (data []*model.TaskEntityModel, err error)
	CountByWorkspace(ctx *abstraction.Context, workspace_id int) (data *int, err error)
	FindByBoardId(ctx *abstraction.Context, board_id int) (*model.TaskEntityModel, error)
	UpdateToNull(ctx *abstraction.Context, data *model.TaskEntityModel, column string) *gorm.DB
}

type task struct {
	abstraction.Repository
}

func NewTask(db *gorm.DB) *task {
	return &task{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *task) Create(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *task) FindById(ctx *abstraction.Context, id int) (*model.TaskEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TaskEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		Preload("CreateBy").
		Preload("UpdateBy").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *task) Update(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *task) FindByBoardIdArr(ctx *abstraction.Context, board_id int) (data []*model.TaskEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "is_delete = @false"+fmt.Sprintf(" AND board_id = %d", board_id))
	limit, offset := general.ProcessLimitOffset(ctx)
	order := general.ProcessOrder(ctx)
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

func (r *task) CountByBoardIdArr(ctx *abstraction.Context, board_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "is_delete = @false"+fmt.Sprintf(" AND board_id = %d", board_id))
	var count model.TaskCountDataModel
	err = r.CheckTrx(ctx).
		Table("task").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task) UpdateCompleted(ctx *abstraction.Context, data *model.TaskEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update("is_completed", data.IsCompleted)
}

func (r *task) Find(ctx *abstraction.Context) (data []*model.TaskEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "is_delete = @false")
	limit, offset := general.ProcessLimitOffset(ctx)
	order := general.ProcessOrder(ctx)
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

func (r *task) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "is_delete = @false")
	var count model.TaskCountDataModel
	err = r.CheckTrx(ctx).
		Table("task").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task) FindByWorkspace(ctx *abstraction.Context, workspace_id int) (data []*model.TaskEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "task.is_delete = @false"+fmt.Sprintf(" AND workspace.id = %d", workspace_id))
	limit, offset := general.ProcessLimitOffset(ctx)
	order := general.ProcessOrder(ctx)
	err = r.CheckTrx(ctx).
		Table("task").
		Joins("JOIN board ON board.id = task.board_id").
		Joins("JOIN workspace ON workspace.id = board.workspace_id").
		Select("task.*").
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

func (r *task) CountByWorkspace(ctx *abstraction.Context, workspace_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task", "task.is_delete = @false"+fmt.Sprintf(" AND workspace.id = %d", workspace_id))
	var count model.TaskCountDataModel
	err = r.CheckTrx(ctx).
		Table("task").
		Joins("JOIN board ON board.id = task.board_id").
		Joins("JOIN workspace ON workspace.id = board.workspace_id").
		Select("COUNT(task.id) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *task) FindByBoardId(ctx *abstraction.Context, board_id int) (*model.TaskEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TaskEntityModel
	err := conn.
		Where("board_id = ? AND is_delete = ?", board_id, false).
		Order("sort_number DESC").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *task) UpdateToNull(ctx *abstraction.Context, data *model.TaskEntityModel, column string) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update(column, nil)
}
