package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type TaskFile interface {
	FindByTaskId(ctx *abstraction.Context, task_id int) (data []*model.TaskFileEntityModel, err error)
	// Create(ctx *abstraction.Context, data *model.TaskFileEntityModel) *gorm.DB
	// FindById(ctx *abstraction.Context, id int) (*model.TaskFileEntityModel, error)
	// Update(ctx *abstraction.Context, data *model.TaskFileEntityModel) *gorm.DB
	// CountByBoardIdArr(ctx *abstraction.Context, board_id int) (data *int, err error)
}

type task_file struct {
	abstraction.Repository
}

func NewTaskFile(db *gorm.DB) *task_file {
	return &task_file{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *task_file) FindByTaskId(ctx *abstraction.Context, task_id int) (data []*model.TaskFileEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "task_file", "is_delete = @false"+fmt.Sprintf(" AND task_id = %d", task_id))
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

// func (r *task_file) Create(ctx *abstraction.Context, data *model.TaskFileEntityModel) *gorm.DB {
// 	return r.CheckTrx(ctx).Create(data)
// }

// func (r *task_file) FindById(ctx *abstraction.Context, id int) (*model.TaskFileEntityModel, error) {
// 	conn := r.CheckTrx(ctx)

// 	var data model.TaskFileEntityModel
// 	err := conn.
// 		Where("id = ? AND is_delete = ?", id, false).
// 		First(&data).
// 		Error
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &data, nil
// }

// func (r *task_file) Update(ctx *abstraction.Context, data *model.TaskFileEntityModel) *gorm.DB {
// 	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
// }

// func (r *task_file) CountByBoardIdArr(ctx *abstraction.Context, board_id int) (data *int, err error) {
// 	where, whereParam := general.ProcessWhereParam(ctx, "task_file", "is_delete = @false"+fmt.Sprintf(" AND board_id = %d", board_id))
// 	var count model.TaskFileCountDataModel
// 	err = r.CheckTrx(ctx).
// 		Table("task_file").
// 		Select("COUNT(*) AS count").
// 		Where(where, whereParam).
// 		Find(&count).
// 		Error
// 	data = &count.Count
// 	return
// }
