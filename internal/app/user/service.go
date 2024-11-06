package contact

import (
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/general"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"

	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Create(ctx *abstraction.Context, payload *dto.UserCreateRequest) (map[string]interface{}, error)
	Find(ctx *abstraction.Context) (map[string]interface{}, error)
	FindById(ctx *abstraction.Context, payload *dto.UserFindByIDRequest) (map[string]interface{}, error)
	Update(ctx *abstraction.Context, payload *dto.UserUpdateRequest) (map[string]interface{}, error)
	Delete(ctx *abstraction.Context, payload *dto.UserDeleteByIDRequest) (map[string]interface{}, error)
	ChangePassword(ctx *abstraction.Context, payload *dto.UserChangePasswordRequest) (map[string]interface{}, error)
	ResetPassword(ctx *abstraction.Context, payload *dto.UserResetPasswordRequest) (map[string]interface{}, error)
}

type service struct {
	UserRepository repository.User

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		UserRepository: f.UserRepository,

		DB: f.Db,
	}
}

func (s *service) Create(ctx *abstraction.Context, payload *dto.UserCreateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userEmail, err := s.UserRepository.FindByEmail(ctx, payload.Email)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userEmail != nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "email already exist")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		modelUser := &model.UserEntityModel{
			Context: ctx,
			UserEntity: model.UserEntity{
				Name:     payload.Name,
				Email:    payload.Email,
				Password: string(hashedPassword),
				RoleId:   payload.RoleId,
				DivisiId: payload.DivisiId,
				IsDelete: false,
				IsLogin:  false,
				IsLocked: false,
			},
		}
		if err = s.UserRepository.Create(ctx, modelUser).Error; err != nil {
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
	data, err := s.UserRepository.Find(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusBadRequest, err, "error find all user")
	}
	count, err := s.UserRepository.Count(ctx)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusBadRequest, err, "error find all user")
	}
	var res []map[string]interface{}
	for _, v := range data {
		res = append(res, map[string]interface{}{
			"id":         v.ID,
			"name":       v.Name,
			"email":      v.Email,
			"created_at": v.CreatedAt,
			"updated_at": v.UpdatedAt,
			"role": map[string]interface{}{
				"id":   v.Role.ID,
				"name": v.Role.Name,
			},
			"divisi": map[string]interface{}{
				"id":   v.Divisi.ID,
				"name": v.Divisi.Name,
			},
		})
	}
	return map[string]interface{}{
		"count": count,
		"data":  res,
	}, nil
}

func (s *service) FindById(ctx *abstraction.Context, payload *dto.UserFindByIDRequest) (map[string]interface{}, error) {
	data, err := s.UserRepository.FindById(ctx, payload.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, response.ErrorBuilder(http.StatusBadRequest, err, "error find by id user")
	}

	res := map[string]interface{}{
		"id":         data.ID,
		"name":       data.Name,
		"email":      data.Email,
		"created_at": data.CreatedAt,
		"updated_at": data.UpdatedAt,
		"role": map[string]interface{}{
			"id":   data.Role.ID,
			"name": data.Role.Name,
		},
		"divisi": map[string]interface{}{
			"id":   data.Divisi.ID,
			"name": data.Divisi.Name,
		},
	}
	return map[string]interface{}{
		"data": res,
	}, nil
}

func (s *service) Update(ctx *abstraction.Context, payload *dto.UserUpdateRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userData, err := s.UserRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user not found")
		}

		newUserData := new(model.UserEntityModel)
		newUserData.Context = ctx
		newUserData.ID = payload.ID
		if payload.Name != nil {
			newUserData.Name = *payload.Name
		}
		if payload.Email != nil {
			userEmail, err := s.UserRepository.FindByEmail(ctx, *payload.Email)
			if err != nil && err.Error() != "record not found" {
				return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
			}
			if userEmail != nil && userEmail.Email != *payload.Email {
				return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "email already exist")
			}
			newUserData.Email = *payload.Email
		}
		if payload.RoleId != nil {
			newUserData.RoleId = *payload.RoleId
		}
		if payload.DivisiId != nil {
			newUserData.DivisiId = *payload.DivisiId
		}

		if err = s.UserRepository.Update(ctx, newUserData).Error; err != nil {
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

func (s *service) Delete(ctx *abstraction.Context, payload *dto.UserDeleteByIDRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userData, err := s.UserRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user not found")
		}

		newUserData := new(model.UserEntityModel)
		newUserData.Context = ctx
		newUserData.ID = payload.ID
		newUserData.IsDelete = true

		if err = s.UserRepository.Update(ctx, newUserData).Error; err != nil {
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

func (s *service) ChangePassword(ctx *abstraction.Context, payload *dto.UserChangePasswordRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userData, err := s.UserRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user not found")
		}

		if err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(payload.OldPassword)); err != nil {
			return response.ErrorBuilder(http.StatusBadRequest, err, "old password is wrong")
		}

		if payload.OldPassword == payload.NewPassword {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "the new password cannot be the same as the old password")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		newUserData := new(model.UserEntityModel)
		newUserData.Context = ctx
		newUserData.ID = payload.ID
		newUserData.Password = string(hashedPassword)

		if err = s.UserRepository.Update(ctx, newUserData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "success change password!",
	}, nil
}

func (s *service) ResetPassword(ctx *abstraction.Context, payload *dto.UserResetPasswordRequest) (map[string]interface{}, error) {
	var new_password string
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userData, err := s.UserRepository.FindById(ctx, payload.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if userData == nil {
			return response.ErrorBuilder(http.StatusBadRequest, errors.New("bad_request"), "user not found")
		}

		passwordString := general.GeneratePassword(8, 1, 1, 1, 1)
		password := []byte(passwordString)
		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		new_password = passwordString

		newUserData := new(model.UserEntityModel)
		newUserData.Context = ctx
		newUserData.ID = payload.ID
		newUserData.Password = string(hashedPassword)

		if err = s.UserRepository.Update(ctx, newUserData).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message":      "success reset password!",
		"new_password": new_password,
	}, nil
}
