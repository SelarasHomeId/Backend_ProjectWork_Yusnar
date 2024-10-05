package affiliate

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
	Create(ctx *abstraction.Context, payload *dto.AffiliateCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context, f *dto.AffiliateFilter, p *abstraction.Pagination) ([]*model.AffiliateEntityModel, *abstraction.PaginationInfo, error)
	FindByID(ctx *abstraction.Context, payload *dto.AffiliateFindByIDRequest) (*model.AffiliateEntityModel, error)
	DeleteByID(ctx *abstraction.Context, payload *dto.AffiliateDeleteByIDRequest) (map[string]interface{}, error)
	Export(ctx *abstraction.Context) (string, *bytes.Buffer, error)
}

type service struct {
	AffiliateRepository    repository.Affiliate
	NotificationRepository repository.Notification

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		AffiliateRepository:    f.AffiliateRepository,
		NotificationRepository: f.NotificationRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.AffiliateCreateRequest) (data map[string]interface{}, err error) {
	if err = trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		modelAffiliate := &model.AffiliateEntityModel{
			Context: ctx,
			AffiliateEntity: model.AffiliateEntity{
				Name:      payload.Name,
				Email:     payload.Email,
				Phone:     payload.Phone,
				Instagram: payload.Instagram,
				Tiktok:    payload.Tiktok,
				CreatedAt: *general.NowLocal(),
			},
		}
		if err = s.AffiliateRepository.Create(ctx, modelAffiliate).Error; err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		if err = s.NotificationRepository.Create(ctx, &model.NotificationEntityModel{
			Context: ctx,
			NotificationEntity: model.NotificationEntity{
				Title:     fmt.Sprintf("Affiliate Request from %s", payload.Name),
				Message:   "Click to see details",
				IsRead:    false,
				CreatedAt: *general.NowLocal(),
				Module:    "affiliate",
				DataID:    fmt.Sprintf("%d", modelAffiliate.ID),
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

func (s *service) Find(ctx *abstraction.Context, f *dto.AffiliateFilter, p *abstraction.Pagination) ([]*model.AffiliateEntityModel, *abstraction.PaginationInfo, error) {
	var (
		data []*model.AffiliateEntityModel
		info *abstraction.PaginationInfo
		err  error
	)
	if data, info, err = s.AffiliateRepository.Find(ctx, f, p); err != nil {
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

func (s *service) FindByID(ctx *abstraction.Context, payload *dto.AffiliateFindByIDRequest) (*model.AffiliateEntityModel, error) {
	var (
		data *model.AffiliateEntityModel
		err  error
	)
	if data, err = s.AffiliateRepository.FindByID(ctx, payload.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	return data, nil
}

func (s *service) DeleteByID(ctx *abstraction.Context, payload *dto.AffiliateDeleteByIDRequest) (data map[string]interface{}, err error) {
	if err = s.AffiliateRepository.DeleteByID(ctx, payload.ID).Error; err != nil {
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
	data, err := s.AffiliateRepository.GetAll(ctx)
	if err != nil {
		return "", nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}

	f := excelize.NewFile()
	sheet := "Data Affiliate Customer"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}
	f.SetActiveSheet(index)
	f.SetCellValue(sheet, "A1", "No")
	f.SetCellValue(sheet, "B1", "Nama")
	f.SetCellValue(sheet, "C1", "Email")
	f.SetCellValue(sheet, "D1", "Telepon")
	f.SetCellValue(sheet, "E1", "Akun Instagram")
	f.SetCellValue(sheet, "F1", "Akun TikTok")
	f.SetCellValue(sheet, "G1", "Tanggal")
	for i, v := range data {
		colA := fmt.Sprintf("A%d", i+2)
		colB := fmt.Sprintf("B%d", i+2)
		colC := fmt.Sprintf("C%d", i+2)
		colD := fmt.Sprintf("D%d", i+2)
		colE := fmt.Sprintf("E%d", i+2)
		colF := fmt.Sprintf("F%d", i+2)
		colG := fmt.Sprintf("G%d", i+2)
		no := i + 1
		f.SetCellValue(sheet, colA, no)
		f.SetCellValue(sheet, colB, v.Name)
		f.SetCellValue(sheet, colC, v.Email)
		f.SetCellValue(sheet, colD, v.Phone)
		f.SetCellValue(sheet, colE, v.Instagram)
		f.SetCellValue(sheet, colF, v.Tiktok)
		f.SetCellValue(sheet, colG, v.CreatedAt)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return "", nil, response.CustomErrorBuilder(http.StatusInternalServerError, err.Error(), "Failed to generate Excel file")
	}
	filename := fmt.Sprintf("ExportData-Affiliate-Customer_%s", general.NowLocal().Format("2006-01-02_15:04:05"))
	return filename, &buf, nil
}
