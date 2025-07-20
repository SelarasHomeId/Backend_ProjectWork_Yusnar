package divisi

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/constant"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"
	"strings"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.DivisiCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.DivisiUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.DivisiDeleteByIDRequest) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.DivisiFindByIDRequest) (map[string]interface{}, error)
	Export(ctx *abstraction.Context) (string, *bytes.Buffer, error)
}

type service struct {
	DivisiRepository    repository.Divisi
	UserRepository      repository.User
	WorkspaceRepository repository.Workspace
	BoardRepository     repository.Board

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		DivisiRepository:    f.DivisiRepository,
		UserRepository:      f.UserRepository,
		WorkspaceRepository: f.WorkspaceRepository,
		BoardRepository:     f.BoardRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.DivisiCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		dataAllDivisi, err := s.DivisiRepository.Find(ctx, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		for _, v := range dataAllDivisi {
			if strings.EqualFold(*payload.Name, v.Name) {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi already exist")
			}
		}

		modelDivisi := &model.DivisiEntityModel{
			Context: ctx,
			DivisiEntity: model.DivisiEntity{
				Name:     *payload.Name,
				IsDelete: false,
			},
		}
		if err := s.DivisiRepository.Create(ctx, modelDivisi).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success create!",
	}, nil
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil
	if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
		return nil, response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
	}
	data, err := s.DivisiRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.DivisiRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"is_delete":  v.IsDelete,
			"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*v.UpdatedAt),
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.DivisiUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		divisiData, err := s.DivisiRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi not found")
		}

		dataAllDivisi, err := s.DivisiRepository.Find(ctx, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newDivisiData := new(model.DivisiEntityModel)
		newDivisiData.Context = ctx
		newDivisiData.ID = payload.ID
		if payload.Name != nil {
			for _, v := range dataAllDivisi {
				if strings.EqualFold(*payload.Name, v.Name) {
					return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi already exist")
				}
			}
			newDivisiData.Name = *payload.Name
		}

		if err = s.DivisiRepository.Update(ctx, newDivisiData).Error; err != nil {
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.DivisiDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if ctx.Auth.RoleID != constant.ROLE_ID_ADMIN {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "this role is not permitted")
		}

		divisiData, err := s.DivisiRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi not found")
		}

		divisiInUserData, err := s.UserRepository.FindByDivisiId(ctx, &payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if divisiInUserData != nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "divisi data is being used")
		}

		newDivisiData := new(model.DivisiEntityModel)
		newDivisiData.Context = ctx
		newDivisiData.ID = divisiData.ID
		newDivisiData.IsDelete = true

		if err = s.DivisiRepository.Update(ctx, newDivisiData).Error; err != nil {
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

func (s *service) FindById(ctx *abstraction.Context, payload *dto.DivisiFindByIDRequest) (map[string]interface{}, error) {
	var res map[string]interface{} = nil
	data, err := s.DivisiRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	if data != nil {
		res = map[string]interface{}{
			"id":         data.ID,
			"name":       data.Name,
			"is_delete":  data.IsDelete,
			"created_at": general.FormatWithZWithoutChangingTime(data.CreatedAt),
			"updated_at": general.FormatWithZWithoutChangingTime(*data.UpdatedAt),
		}
	}
	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Export(ctx *abstraction.Context) (string, *bytes.Buffer, error) {
	data, err := s.DivisiRepository.Find(ctx, true)
	if err != nil && err.Error() != "record not found" {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	f := excelize.NewFile()
	sheet := "Master Data - Divisi"
	index, err := f.NewSheet(general.TruncateSheetName(sheet))
	if err != nil {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)
	f.SetCellValue(sheet, "A1", "No")
	f.SetCellValue(sheet, "B1", "Nama")
	f.SetCellValue(sheet, "C1", "Tanggal Dibuat")
	for i, v := range data {
		colA := fmt.Sprintf("A%d", i+2)
		colB := fmt.Sprintf("B%d", i+2)
		colC := fmt.Sprintf("C%d", i+2)
		no := i + 1
		f.SetCellValue(sheet, colA, no)
		f.SetCellValue(sheet, colB, v.Name)
		f.SetCellValue(sheet, colC, v.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	filename := fmt.Sprintf("Master Data - Divisi (%s).xlsx", general.NowLocal().Format("2006-01-02"))
	return filename, &buf, nil
}
