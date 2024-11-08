package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type Divisi interface {
	FindById(ctx *abstraction.Context, id int) (*model.DivisiEntityModel, error)
}

type divisi struct {
	abstraction.Repository
}

func NewDivisi(db *gorm.DB) *divisi {
	return &divisi{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *divisi) FindById(ctx *abstraction.Context, id int) (*model.DivisiEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.DivisiEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}
