package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Board interface {
	Create(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB
	FindByWorkspaceId(ctx *abstraction.Context, workspace_id int) (*model.BoardEntityModel, error)
	FindById(ctx *abstraction.Context, id int) (*model.BoardEntityModel, error)
	Update(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB
	FindByWorkspaceIdArr(ctx *abstraction.Context, workspace_id int) (data []*model.BoardEntityModel, err error)
	CountByWorkspaceIdArr(ctx *abstraction.Context, workspace_id int) (data *int, err error)
	UpdateTaskTotalById(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB
}

type board struct {
	abstraction.Repository
}

func NewBoard(db *gorm.DB) *board {
	return &board{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *board) Create(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *board) FindByWorkspaceId(ctx *abstraction.Context, workspace_id int) (*model.BoardEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BoardEntityModel
	err := conn.
		Where("workspace_id = ? AND is_delete = ?", workspace_id, false).
		Order("sort_number DESC").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *board) FindById(ctx *abstraction.Context, id int) (*model.BoardEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.BoardEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *board) Update(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *board) FindByWorkspaceIdArr(ctx *abstraction.Context, workspace_id int) (data []*model.BoardEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "board", "is_delete = @false"+fmt.Sprintf(" AND workspace_id = %d", workspace_id))
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

func (r *board) CountByWorkspaceIdArr(ctx *abstraction.Context, workspace_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "board", "is_delete = @false"+fmt.Sprintf(" AND workspace_id = %d", workspace_id))
	var count model.BoardCountDataModel
	err = r.CheckTrx(ctx).
		Table("board").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *board) UpdateTaskTotalById(ctx *abstraction.Context, data *model.BoardEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update("task_total", data.TaskTotal)
}
