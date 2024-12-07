package banner

import (
	"errors"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/response"

	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
}

type service struct {
	BannerRepository repository.Banner
	DB               *gorm.DB
	sDrive           *drive.Service
	fDrive           *drive.File
}

func NewService(f *factory.Factory) Service {
	return &service{
		BannerRepository: f.BannerRepository,
		DB:               f.Db,
		sDrive:           f.GDrive.Service,
		fDrive:           f.GDrive.Folder,
	}
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{}
	data, err := s.BannerRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.BannerRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		fileDrive, err := gdrive.GetFile(s.sDrive, v.FileId)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
		}
		res = append(res, map[string]interface{}{
			"id": v.ID,
			"file": map[string]interface{}{
				"content": fileDrive.WebContentLink,
				"view":    fileDrive.WebViewLink,
			},
			"file_name":  v.FileName,
			"is_delete":  v.IsDelete,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}
