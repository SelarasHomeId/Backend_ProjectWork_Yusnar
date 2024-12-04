package repository

import (
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/model"

	"gorm.io/gorm"
)

type Test interface {
	Create(ctx *abstraction.Context, data *model.TestEntityModel) *gorm.DB
	FindById(ctx *abstraction.Context, id int) (*model.TestEntityModel, error)
}

type test struct {
	abstraction.Repository
}

func NewTest(db *gorm.DB) *test {
	return &test{
		Repository: abstraction.Repository{
			Db: db,
		},
	}
}

func (r *test) Create(ctx *abstraction.Context, data *model.TestEntityModel) *gorm.DB {
	return r.CheckTrx(ctx).Create(data)
}

func (r *test) FindById(ctx *abstraction.Context, id int) (*model.TestEntityModel, error) {
	conn := r.CheckTrx(ctx)

	var data model.TestEntityModel
	err := conn.
		Where("id = ? AND is_delete = ?", id, false).
		First(&data).
		Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}
