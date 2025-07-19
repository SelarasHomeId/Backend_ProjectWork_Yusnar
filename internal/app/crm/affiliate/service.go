package affiliate

import (
	"bytes"
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

	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.AffiliateCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	Export(ctx *abstraction.Context) (string, *bytes.Buffer, error)
}

type service struct {
	AffiliateRepository  repository.Affiliate
	NotifikasiRepository repository.Notifikasi
	UserRepository       repository.User

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		AffiliateRepository:  f.AffiliateRepository,
		NotifikasiRepository: f.NotifikasiRepository,
		UserRepository:       f.UserRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.AffiliateCreateRequest) (data map[string]interface{}, err error) {
	var modelAffiliateBackup *model.AffiliateEntityModel
	if err = trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userAdmin, err := s.UserRepository.FindByRoleIdArr(ctx, constant.ROLE_ID_ADMIN, true)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		modelAffiliate := &model.AffiliateEntityModel{
			Context: ctx,
			AffiliateEntity: model.AffiliateEntity{
				Name:      *payload.Name,
				Email:     *payload.Email,
				Phone:     *payload.Phone,
				Instagram: *payload.Instagram,
				Tiktok:    *payload.Tiktok,
				Info:      *payload.Info,
			},
		}
		if err := s.AffiliateRepository.Create(ctx, modelAffiliate).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		} else {
			modelAffiliateBackup = modelAffiliate
		}

		for _, v := range userAdmin {
			modelNotifikasi := new(model.NotifikasiEntityModel)
			modelNotifikasi.Context = ctx
			modelNotifikasi.Title = "Data affiliator baru telah masuk"
			modelNotifikasi.Message = fmt.Sprintf("%s - %s", *payload.Name, *payload.Info)
			modelNotifikasi.IsRead = false
			modelNotifikasi.UserId = v.ID
			modelNotifikasi.TaskId = constant.BLANK_TASK_ID
			if err := s.NotifikasiRepository.Create(ctx, modelNotifikasi).Error; err != nil {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
		}

		return nil
	}); err != nil {
		if modelAffiliateBackup != nil {
			s.AffiliateRepository.CreateWithoutContext(ctx, modelAffiliateBackup)
			logrus.Info("Recovery insert affiliate data success")
		}
		return nil, err
	}
	return map[string]interface{}{
		"message": "success",
	}, nil
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{} = nil
	data, err := s.AffiliateRepository.Find(ctx, false)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	count, err := s.AffiliateRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"phone":      v.Phone,
			"email":      v.Email,
			"instagram":  v.Instagram,
			"tiktok":     v.Tiktok,
			"info":       v.Info,
			"created_at": general.FormatWithZWithoutChangingTime(v.CreatedAt),
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) Export(ctx *abstraction.Context) (string, *bytes.Buffer, error) {
	data, err := s.AffiliateRepository.Find(ctx, true)
	if err != nil && err.Error() != "record not found" {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}

	f := excelize.NewFile()
	sheet := "Affiliate Request"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	f.DeleteSheet("Sheet1")
	f.SetActiveSheet(index)
	f.SetCellValue(sheet, "A1", "No")
	f.SetCellValue(sheet, "B1", "Nama")
	f.SetCellValue(sheet, "C1", "Email")
	f.SetCellValue(sheet, "D1", "Telepon")
	f.SetCellValue(sheet, "E1", "Instagram")
	f.SetCellValue(sheet, "F1", "Tiktok")
	f.SetCellValue(sheet, "G1", "Info")
	f.SetCellValue(sheet, "H1", "Tanggal")
	for i, v := range data {
		colA := fmt.Sprintf("A%d", i+2)
		colB := fmt.Sprintf("B%d", i+2)
		colC := fmt.Sprintf("C%d", i+2)
		colD := fmt.Sprintf("D%d", i+2)
		colE := fmt.Sprintf("E%d", i+2)
		colF := fmt.Sprintf("F%d", i+2)
		colG := fmt.Sprintf("G%d", i+2)
		colH := fmt.Sprintf("H%d", i+2)
		no := i + 1
		f.SetCellValue(sheet, colA, no)
		f.SetCellValue(sheet, colB, v.Name)
		f.SetCellValue(sheet, colC, v.Email)
		f.SetCellValue(sheet, colD, v.Phone)
		f.SetCellValue(sheet, colE, v.Instagram)
		f.SetCellValue(sheet, colF, v.Tiktok)
		f.SetCellValue(sheet, colG, v.Info)
		f.SetCellValue(sheet, colH, v.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return "", nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	filename := fmt.Sprintf("Affiliate Request (%s).xlsx", general.NowLocal().Format("2006-01-02"))
	return filename, &buf, nil
}
