package notifikasi

import (
	"errors"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"gorm.io/gorm"
)

type Service interface {
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	SetRead(ctx *abstraction.Context, payload *dto.NotifikasiSetReadRequest) (map[string]interface{}, error)
}

type service struct {
	NotifikasiRepository repository.Notifikasi

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		NotifikasiRepository: f.NotifikasiRepository,

		DB: f.Db,
	}
}

func (s *service) Find(ctx *abstraction.Context) (map[string]interface{}, error) {
	var res []map[string]interface{}
	data, err := s.NotifikasiRepository.FindByUserId(ctx, &ctx.Auth.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	countTotal, countRead, countUnread, err := s.NotifikasiRepository.CountByUserId(ctx, &ctx.Auth.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
	}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"title":      v.Title,
			"message":    v.Message,
			"is_read":    v.IsRead,
			"user_id":    v.UserId,
			"link":       v.Link,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
		})
	}
	return map[string]interface{}{
		"count_total":  countTotal,
		"count_read":   countRead,
		"count_unread": countUnread,
		"data":         res,
	}, nil
}

func (s *service) SetRead(ctx *abstraction.Context, payload *dto.NotifikasiSetReadRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		notifikasiData, err := s.NotifikasiRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if notifikasiData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "notifikasi not found")
		}

		newNotifikasiData := new(model.NotifikasiEntityModel)
		newNotifikasiData.Context = ctx
		newNotifikasiData.ID = notifikasiData.ID
		newNotifikasiData.IsRead = true

		if err = s.NotifikasiRepository.Update(ctx, newNotifikasiData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success set read!",
	}, nil
}
