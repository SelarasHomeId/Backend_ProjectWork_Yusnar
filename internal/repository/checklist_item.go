package repository

import (
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type ChecklistItem interface {
	Create(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB
	FindByTaskChecklistId(ctx *abstraction.Context, task_checklist_id int) (*model.ChecklistItemEntityModel, error)
	FindByTaskChecklistIdArr(ctx *abstraction.Context, task_checklist_id int) (data []*model.ChecklistItemEntityModel, err error)
	CountByTaskChecklistIdArr(ctx *abstraction.Context, task_checklist_id int) (data *int, err error)
	Update(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB
	FindById(ctx *abstraction.Context, id int) (*model.ChecklistItemEntityModel, error)
	UpdateToNull(ctx *abstraction.Context, data *model.ChecklistItemEntityModel, column string) *gorm.DB
	UpdateCompleted(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB
}

type checklist_item struct {
	abstraction.Repository
}

func NewChecklistItem(db *gorm.DB) *checklist_item {
	return &checklist_item{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *checklist_item) Create(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *checklist_item) FindByTaskChecklistId(ctx *abstraction.Context, task_checklist_id int) (*model.ChecklistItemEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.ChecklistItemEntityModel
	err := conn.
		Where("task_checklist_id = ? AND is_delete = ?", task_checklist_id, false).
		Order("sort_number DESC").
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *checklist_item) FindByTaskChecklistIdArr(ctx *abstraction.Context, task_checklist_id int) (data []*model.ChecklistItemEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "checklist_item", "is_delete = @false"+fmt.Sprintf(" AND task_checklist_id = %d", task_checklist_id))
	limit, offset := general.ProcessLimitOffset(ctx)
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

func (r *checklist_item) CountByTaskChecklistIdArr(ctx *abstraction.Context, task_checklist_id int) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "checklist_item", "is_delete = @false"+fmt.Sprintf(" AND task_checklist_id = %d", task_checklist_id))
	var count model.ChecklistItemCountDataModel
	err = r.CheckTrx(ctx).
		Table("checklist_item").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *checklist_item) Update(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Updates(data)
}

func (r *checklist_item) FindById(ctx *abstraction.Context, id int) (*model.ChecklistItemEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.ChecklistItemEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *checklist_item) UpdateToNull(ctx *abstraction.Context, data *model.ChecklistItemEntityModel, column string) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update(column, nil)
}

func (r *checklist_item) UpdateCompleted(ctx *abstraction.Context, data *model.ChecklistItemEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("id = ?", data.ID).Update("is_completed", data.IsCompleted)
}
