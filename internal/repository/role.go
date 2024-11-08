package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type Role interface {
	FindById(ctx *abstraction.Context, id int) (*model.RoleEntityModel, error)
}

type role struct {
	abstraction.Repository
}

func NewRole(db *gorm.DB) *role {
	return &role{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *role) FindById(ctx *abstraction.Context, id int) (*model.RoleEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.RoleEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}
