package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"
	"selarashomeid/pkg/util/general"

	"gorm.io/gorm"
)

type Workspace interface {
	Create(ctx *abstraction.Context, data *model.WorkspaceEntityModel) *gorm.DB
	UpdateByDivisiId(ctx *abstraction.Context, data *model.WorkspaceEntityModel) *gorm.DB
	FindByDivisiId(ctx *abstraction.Context, divisi_id int) (*model.WorkspaceEntityModel, error)
	Find(ctx *abstraction.Context) (data []*model.WorkspaceEntityModel, err error)
	Count(ctx *abstraction.Context) (data *int, err error)
	FindById(ctx *abstraction.Context, id int) (*model.WorkspaceEntityModel, error)
}

type workspace struct {
	abstraction.Repository
}

func NewWorkspace(db *gorm.DB) *workspace {
	return &workspace{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *workspace) Create(ctx *abstraction.Context, data *model.WorkspaceEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *workspace) UpdateByDivisiId(ctx *abstraction.Context, data *model.WorkspaceEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Model(data).Where("divisi_id = ?", data.DivisiId).Updates(data)
}

func (r *workspace) FindByDivisiId(ctx *abstraction.Context, divisi_id int) (*model.WorkspaceEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.WorkspaceEntityModel
	err := conn.
		Where("divisi_id = ? AND is_delete = ?", divisi_id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *workspace) Find(ctx *abstraction.Context) (data []*model.WorkspaceEntityModel, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "workspace", "is_delete = @false")
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

func (r *workspace) Count(ctx *abstraction.Context) (data *int, err error) {
	where, whereParam := general.ProcessWhereParam(ctx, "workspace", "is_delete = @false")
	var count model.WorkspaceCountDataModel
	err = r.CheckTrx(ctx).
		Table("workspace").
		Select("COUNT(*) AS count").
		Where(where, whereParam).
		Find(&count).
		Error
	data = &count.Count
	return
}

func (r *workspace) FindById(ctx *abstraction.Context, id int) (*model.WorkspaceEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.WorkspaceEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}
