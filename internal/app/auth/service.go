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
	modeltoken "selarashomeid/internal/model/token"
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
	Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (*dto.AuthLoginResponse, error)
	Logout(ctx *abstraction.Context) (map[string]interface{}, error)
	ChangePassword(ctx *abstraction.Context, payload *dto.ChangePasswordRequest) (map[string]interface{}, error)
	ResetPassword(ctx *abstraction.Context, payload *dto.ResetPasswordRequest) (map[string]interface{}, error)
}

type service struct {
	AdminRepository repository.Admin

	DB *gorm.DB
}

func NewService(f *factory.Factory) Service {
	return &service{
		AdminRepository: f.AdminRepository,

		DB: f.Db,
	}
}

func (s *service) encryptTokenClaims(v int) (encryptedString string, err error) {
	encryptedString, err = aescrypt.EncryptAES(fmt.Sprint(v), config.Get().JWT.SecretKey)
	return
}

func (s *service) Login(ctx *abstraction.Context, payload *dto.AuthLoginRequest) (*dto.AuthLoginResponse, error) {
	data, err := s.AdminRepository.FindByUsername(ctx, payload.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.Unauthorized, errors.New("username or password is incorrect"))
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(data.Password), []byte(payload.Password)); err != nil {
		return nil, response.ErrorBuilder(&response.ErrorConstant.Unauthorized, errors.New("username or password is incorrect"))
	}

	var encryptedUserID string
	if encryptedUserID, err = s.encryptTokenClaims(data.ID); err != nil {
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}

	encodedUsername := encoding.Encode(data.Username)
	encodedEmail := encoding.Encode(data.Email)

	tokenClaims := &modeltoken.TokenClaims{
		ID:       encryptedUserID,
		Username: encodedUsername,
		Email:    encodedEmail,
		Exp:      time.Now().Add(time.Duration(24 * time.Hour)).Unix(),
	}
	authToken := modeltoken.NewAuthToken(tokenClaims)
	token, err := authToken.Token()
	if err != nil {
		return nil, response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
	}

	if data.IsLogin {
		return nil, response.ErrorBuilder(&response.ErrorConstant.Unauthorized, errors.New("user already login"))
	} else {
		s.AdminRepository.UpdateLoginTrue(ctx, data.ID)
	}

	return &dto.AuthLoginResponse{
		Token: token,
		AdminEntityModel: model.AdminEntityModel{
			ID: data.ID,
			AdminEntity: model.AdminEntity{
				Name:     data.Name,
				Email:    data.Email,
				Username: data.Username,
			},
		},
	}, nil
}

func (s *service) Logout(ctx *abstraction.Context) (map[string]interface{}, error) {
	s.AdminRepository.UpdateLoginFalse(ctx, ctx.Auth.ID)
	return map[string]interface{}{
		"message": "success",
	}, nil
}

func (s *service) ChangePassword(ctx *abstraction.Context, payload *dto.ChangePasswordRequest) (map[string]interface{}, error) {
	if payload.Id != ctx.Auth.ID {
		return nil, response.CustomErrorBuilder(http.StatusNotAcceptable, "failed change password", "your request id is not matching!")
	}
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		userData, err := s.AdminRepository.FindById(ctx, payload.Id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.ErrorBuilder(&response.ErrorConstant.Unauthorized, errors.New("admin not found"))
			}
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		if err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(payload.OldPassword)); err != nil {
			return response.CustomErrorBuilder(http.StatusBadRequest, "Request Failed", "Your password is wrong!")
		}
		password := []byte(payload.NewPassword)
		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}

		if err := s.AdminRepository.Update(ctx, &model.AdminEntityModel{
			Context: ctx,
			AdminEntity: model.AdminEntity{
				Password: string(hashedPassword),
			},
			ID: payload.Id,
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

func (s *service) ResetPassword(ctx *abstraction.Context, payload *dto.ResetPasswordRequest) (map[string]interface{}, error) {
	if err := trxmanager.New(s.DB).WithTrx(ctx, func(ctx *abstraction.Context) error {
		newPassword := "Test12345*"
		password := []byte(newPassword)
		hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}

		if err := s.AdminRepository.Update(ctx, &model.AdminEntityModel{
			Context: ctx,
			AdminEntity: model.AdminEntity{
				Password: string(hashedPassword),
			},
			ID: payload.Id,
		}).Error; err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "Your new password: Test12345*",
	}, nil
}
