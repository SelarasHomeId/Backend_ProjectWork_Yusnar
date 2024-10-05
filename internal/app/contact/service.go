package contact

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.ContactCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context, f *dto.ContactFilter, p *abstraction.Pagination) ([]*model.ContactEntityModel, *abstraction.PaginationInfo, error)
	FindByID(ctx *abstraction.Context, payload *dto.ContactFindByIDRequest) (*model.ContactEntityModel, error)
	DeleteByID(ctx *abstraction.Context, payload *dto.ContactDeleteByIDRequest) (map[string]interface{}, error)
	Export(ctx *abstraction.Context) (string, *bytes.Buffer, error)
}

type service struct {
	ContactRepository      repository.Contact
	NotificationRepository repository.Notification

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		ContactRepository:      f.ContactRepository,
		NotificationRepository: f.NotificationRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.ContactCreateRequest) (data map[string]interface{}, err error) {
	if err = trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		modelContact := &model.ContactEntityModel{
			Context: ctx,
			ContactEntity: model.ContactEntity{
				Name:      payload.Name,
				Email:     payload.Email,
				Phone:     payload.Phone,
				Message:   payload.Message,
				CreatedAt: *general.NowLocal(),
			},
		}
		if err = s.ContactRepository.Create(ctx, modelContact).Error; err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		if err = s.NotificationRepository.Create(ctx, &model.NotificationEntityModel{
			Context: ctx,
			NotificationEntity: model.NotificationEntity{
				Title:     fmt.Sprintf("Customer (%s) give your message", payload.Name),
				Message:   "Click to see details",
				IsRead:    false,
				CreatedAt: *general.NowLocal(),
				Module:    "contact",
				DataID:    fmt.Sprintf("%d", modelContact.ID),
			},
		}).Error; err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success",
	}, nil
}

func (s *service) Find(ctx *abstraction.Context, f *dto.ContactFilter, p *abstraction.Pagination) ([]*model.ContactEntityModel, *abstraction.PaginationInfo, error) {
	var (
		data []*model.ContactEntityModel
		info *abstraction.PaginationInfo
		err  error
	)
	if data, info, err = s.ContactRepository.Find(ctx, f, p); err != nil {
		return nil, nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	if p != nil && p.PageSize != nil {
		info.Pages = int(math.Ceil(float64(info.Count) / float64(*p.PageSize)))
		if len(data) > *p.PageSize {
			data = data[:len(data)-1]
			info.MoreRecords = true
		}
	}
	return data, info, nil
}

func (s *service) FindByID(ctx *abstraction.Context, payload *dto.ContactFindByIDRequest) (*model.ContactEntityModel, error) {
	var (
		data *model.ContactEntityModel
		err  error
	)
	if data, err = s.ContactRepository.FindByID(ctx, payload.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	return data, nil
}

func (s *service) DeleteByID(ctx *abstraction.Context, payload *dto.ContactDeleteByIDRequest) (data map[string]interface{}, err error) {
	if err = s.ContactRepository.DeleteByID(ctx, payload.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	if err = s.NotificationRepository.DeleteByDataID(ctx, payload.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	return map[string]interface{}{
		"message": "success",
	}, nil
}

func (s *service) Export(ctx *abstraction.Context) (string, *bytes.Buffer, error) {
	data, err := s.ContactRepository.GetAll(ctx)
	if err != nil {
		return "", nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}

	f := excelize.NewFile()
	sheet := "Data Contact Customer"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	f.SetActiveSheet(index)
	f.SetCellValue(sheet, "A1", "No")
	f.SetCellValue(sheet, "B1", "Nama")
	f.SetCellValue(sheet, "C1", "Email")
	f.SetCellValue(sheet, "D1", "Telepon")
	f.SetCellValue(sheet, "E1", "Pesan")
	f.SetCellValue(sheet, "F1", "Tanggal")
	for i, v := range data {
		colA := fmt.Sprintf("A%d", i+2)
		colB := fmt.Sprintf("B%d", i+2)
		colC := fmt.Sprintf("C%d", i+2)
		colD := fmt.Sprintf("D%d", i+2)
		colE := fmt.Sprintf("E%d", i+2)
		colF := fmt.Sprintf("F%d", i+2)
		no := i + 1
		f.SetCellValue(sheet, colA, no)
		f.SetCellValue(sheet, colB, v.Name)
		f.SetCellValue(sheet, colC, v.Email)
		f.SetCellValue(sheet, colD, v.Phone)
		f.SetCellValue(sheet, colE, v.Message)
		f.SetCellValue(sheet, colF, v.CreatedAt)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return "", nil, response.CustomErrorBuilder(http.StatusInternalServerError, err.Error(), "Failed to generate Excel file")
	}
	filename := fmt.Sprintf("ExportData-Contact-Customer_%s", general.NowLocal().Format("2006-01-02_15:04:05"))
	return filename, &buf, nil
}
