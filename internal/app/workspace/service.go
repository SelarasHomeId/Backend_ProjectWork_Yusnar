package workspace

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/response"

	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	WorkspaceRepository repository.Workspace

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		WorkspaceRepository: f.WorkspaceRepository,

		DB: f.Db,
	}
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var (
		res   []map[string]interface{}
		count *int
	)

	if ctx.Auth.DivisiID != constant.DIVISI_ID_MANAGEMENT {
		countData := 1
		data, err := s.WorkspaceRepository.FindByDivisiId(ctx, ctx.Auth.DivisiID)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		res = append(res, map[string]interface{}{
			"id":         data.ID,
			"divisi_id":  data.DivisiId,
			"name":       data.Name,
			"is_delete":  data.IsDelete,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		})
		count = &countData
	} else {
		data, err := s.WorkspaceRepository.Find(ctx)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		count, err = s.WorkspaceRepository.Count(ctx)
		if err != nil && err.Error() != "record not found" {
			return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		for _, v := range data {
			res = append(res, map[string]interface{}{
				"id":         v.ID,
				"divisi_id":  v.DivisiId,
				"name":       v.Name,
				"is_delete":  v.IsDelete,
				"created_at": v.CreatedAt,
				"updated_at": v.UpdatedAt,
			})
		}
	}

	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
