package auth

import (
	"errors"
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/config"
	"selarashomeid/internal/dto"
	"selarashomeid/internal/factory"
	"selarashomeid/internal/model"
	modelToken "selarashomeid/internal/model/token"
	"selarashomeid/internal/repository"
	"selarashomeid/pkg/util/aescrypt"
	"selarashomeid/pkg/util/encoding"
	"selarashomeid/pkg/util/response"
	"selarashomeid/pkg/util/trxmanager"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (map[string]interface{}, error)
	Logout(ctx *abstraction.Context) (map[string]interface{}, error)
	RefreshToken(ctx *abstraction.Context) (map[string]interface{}, error)
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

func (s *service) encryptTokenClaims(v int) (encryptedString string, err error) {
	encryptedString, err = aescrypt.EncryptAES(fmt.Sprint(v), config.Get().JWT.SecretKey)
	return
}

func (s *service) Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (map[string]interface{}, error) {
	var (
		err   error
		data  = new(model.UserEntityModel)
		token string
	)
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		data, err = s.UserRepository.FindByEmail(ctx, payload.Email)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if data == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "email or password is incorrect")
		}

		if err = bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(payload.Password)); err != nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "email or password is incorrect")
		}

		if data.IsLocked {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "this account is locked")
		}

		if data.IsLogin {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "user already login")
		}

		var encryptedUserID string
		if encryptedUserID, err = s.encryptTokenClaims(data.ID); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserRoleID string
		if encryptedUserRoleID, err = s.encryptTokenClaims(data.RoleId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserDivisiID string
		if encryptedUserDivisiID, err = s.encryptTokenClaims(data.DivisiId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		encodedEmail := encoding.Encode(data.Email)

		tokenClaims := &modelToken.TokenClaims{
			ID:       encryptedUserID,
			RoleID:   encryptedUserRoleID,
			DivisiID: encryptedUserDivisiID,
			Email:    encodedEmail,
			Exp:      time.Now().Add(time.Duration(1 * time.Hour)).Unix(),
		}
		authToken := modelToken.NewAuthToken(tokenClaims)
		token, err = authToken.Token()
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		if err := s.UserRepository.UpdateLogin(ctx, &data.ID, true).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if err := s.UserRepository.UpdateLoginFrom(ctx, &data.ID, payload.LoginFrom).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
		"data": map[string]interface{}{
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
		},
	}, nil
}

func (s *service) Logout(ctx *abstraction.Context) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		if err := s.UserRepository.UpdateLogin(ctx, &ctx.Auth.ID, false).Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		if err := s.UserRepository.UpdateLoginFrom(ctx, &ctx.Auth.ID, "").Error; err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message": "success logout!",
	}, nil
}

func (s *service) RefreshToken(ctx *abstraction.Context) (map[string]interface{}, error) {
	var token string
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		data, err := s.UserRepository.FindById(ctx, ctx.Auth.ID)
		if err != nil && err.Error() != "record not found" {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		var encryptedUserID string
		if encryptedUserID, err = s.encryptTokenClaims(data.ID); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserRoleID string
		if encryptedUserRoleID, err = s.encryptTokenClaims(data.RoleId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		var encryptedUserDivisiID string
		if encryptedUserDivisiID, err = s.encryptTokenClaims(data.DivisiId); err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}
		encodedEmail := encoding.Encode(data.Email)

		tokenClaims := &modelToken.TokenClaims{
			ID:       encryptedUserID,
			RoleID:   encryptedUserRoleID,
			DivisiID: encryptedUserDivisiID,
			Email:    encodedEmail,
			Exp:      time.Now().Add(time.Duration(1 * time.Hour)).Unix(),
		}
		authToken := modelToken.NewAuthToken(tokenClaims)
		token, err = authToken.Token()
		if err != nil {
			return response.ErrorBuilder(http.StatusInternalServerError, err, "server_error")
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
	}, nil
}
