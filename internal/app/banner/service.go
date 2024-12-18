package banner

import (
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/gdrive"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"google.golang.org/api/drive/v3"
	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.BannerFindByIDRequest) (map[string]interface{}, error)
	Create(ctx *abstraction.Context, payload *dto.BannerCreateRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.BannerUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.BannerDeleteByIDRequest) (map[string]interface{}, error)
	GetPopup(ctx *abstraction.Context) (map[string]interface{}, error)
	UpdatePopup(ctx *abstraction.Context, payload *dto.BannerUpdatePopupRequest) (map[string]interface{}, error)
	SetPopup(ctx *abstraction.Context, payload *dto.BannerSetPopupRequest) (map[string]interface{}, error)
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
				"view":    "https://lh3.googleusercontent.com/d/" + v.FileId,
				"content": fileDrive.WebContentLink,
			},
			"file_name":  v.FileName,
			"is_delete":  v.IsDelete,
			"is_popup":   v.IsPopup,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) FindById(ctx *abstraction.Context, payload *dto.BannerFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil
	data, err := s.BannerRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		fileDrive, err := gdrive.GetFile(s.sDrive, data.FileId)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
		}
		res = map[string]interface{}{
			"id": data.ID,
			"file": map[string]interface{}{
				"view":    "https://lh3.googleusercontent.com/d/" + data.FileId,
				"content": fileDrive.WebContentLink,
			},
			"file_name":  data.FileName,
			"is_delete":  data.IsDelete,
			"is_popup":   data.IsPopup,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		}

	}
	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.BannerCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		for _, file := range payload.Files {
			f, err := file.Open()
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			defer f.Close()

			isImageFile, fullFileName := general.ValidateImage(file.Filename)
			if !isImageFile {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), fmt.Sprintf("file format for %s is not approved", file.Filename))
			}

			newFile, err := gdrive.CreateFile(s.sDrive, fullFileName, "application/octet-stream", f, s.fDrive.Id)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			modelBanner := &model.BannerEntityModel{
				Context: ctx,
				BannerEntity: model.BannerEntity{
					FileId:   newFile.Id,
					FileName: newFile.Name,
					IsDelete: false,
					IsPopup:  false,
				},
			}
			if err := s.BannerRepository.Create(ctx, modelBanner).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success create!",
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.BannerUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		bannerData, err := s.BannerRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if bannerData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "banner not found")
		}

		newbannerData := new(model.BannerEntityModel)
		newbannerData.Context = ctx
		newbannerData.ID = payload.ID
		if payload.FileName != nil {
			newbannerData.FileName = *payload.FileName
		}
		if payload.Files != nil {
			file := payload.Files[0]

			f, err := file.Open()
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			defer f.Close()

			isImageFile, fullFileName := general.ValidateImage(file.Filename)
			if !isImageFile {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), fmt.Sprintf("file format for %s is not approved", file.Filename))
			}

			newFile, err := gdrive.CreateFile(s.sDrive, fullFileName, "application/octet-stream", f, s.fDrive.Id)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newbannerData.FileId = newFile.Id

			err = gdrive.DeleteFile(s.sDrive, bannerData.FileId)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		if err = s.BannerRepository.Update(ctx, newbannerData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success update!",
	}, nil
}

func (s *service) Delete(ctx *abstraction.Context, payload *dto.BannerDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		bannerData, err := s.BannerRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if bannerData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "banner not found")
		}

		newbannerData := new(model.BannerEntityModel)
		newbannerData.Context = ctx
		newbannerData.ID = bannerData.ID
		newbannerData.IsDelete = true

		if err = s.BannerRepository.Update(ctx, newbannerData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success delete!",
	}, nil
}

func (s *service) GetPopup(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res map[string]interface{} = nil
	data, err := s.BannerRepository.GetPopup(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		fileDrive, err := gdrive.GetFile(s.sDrive, data.FileId)
		if err != nil {
			return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "file not found")
		}
		res = map[string]interface{}{
			"id": data.ID,
			"file": map[string]interface{}{
				"view":    "https://lh3.googleusercontent.com/d/" + data.FileId,
				"content": fileDrive.WebContentLink,
			},
			"file_name":  data.FileName,
			"is_delete":  data.IsDelete,
			"is_popup":   data.IsPopup,
			"created_at": data.CreatedAt,
			"updated_at": data.UpdatedAt,
		}

	}
	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) UpdatePopup(ctx *abstraction.Context, payload *dto.BannerUpdatePopupRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		bannerData, err := s.BannerRepository.FindByIdAndPopupTrue(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if bannerData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "popup not found")
		}

		newbannerData := new(model.BannerEntityModel)
		newbannerData.Context = ctx
		newbannerData.ID = payload.ID
		if payload.FileName != nil {
			newbannerData.FileName = *payload.FileName
		}
		if payload.Files != nil {
			file := payload.Files[0]

			f, err := file.Open()
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			defer f.Close()

			isImageFile, fullFileName := general.ValidateImage(file.Filename)
			if !isImageFile {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), fmt.Sprintf("file format for %s is not approved", file.Filename))
			}

			newFile, err := gdrive.CreateFile(s.sDrive, fullFileName, "application/octet-stream", f, s.fDrive.Id)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}

			newbannerData.FileId = newFile.Id

			err = gdrive.DeleteFile(s.sDrive, bannerData.FileId)
			if err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		if err = s.BannerRepository.Update(ctx, newbannerData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success update!",
	}, nil
}

func (s *service) SetPopup(ctx *abstraction.Context, payload *dto.BannerSetPopupRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		bannerData, err := s.BannerRepository.FindByPopupTrue(ctx)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if bannerData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "banner not found")
		}

		var set bool
		if payload.Set == "true" {
			set = false
		} else if payload.Set == "false" {
			set = true
		} else {
			set = false
		}

		newbannerData := new(model.BannerEntityModel)
		newbannerData.Context = ctx
		newbannerData.IsDelete = set
		newbannerData.IsPopup = true

		if err = s.BannerRepository.UpdateByPopupTrue(ctx, newbannerData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": fmt.Sprintf("success set popup %s", payload.Set),
	}, nil
}
