package dto

import (
	"fmt"
	"selarashomeid/internal/config"
	"selarashomeid/internal/model"
	modeltoken "selarashomeid/internal/model/token"

	"github.com/golang-jwt/jwt"
)

type AuthLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthLoginResponse struct {
	Token string `json:"token"`
	model.AdminEntityModel
}

type RefreshTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

func (r RefreshTokenRequest) TokenClaims() (*modeltoken.TokenClaims, error) {
	token, err := jwt.Parse(r.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method :%v", token.Header["alg"])
		}
		return []byte(config.Get().JWT.SecretKey), nil
	})
	if token == nil || !token.Valid || err != nil {
		if jwtErrValidation, ok := err.(*jwt.ValidationError); ok {
			c := token.Claims.(jwt.MapClaims)
			return &modeltoken.TokenClaims{
				ID:       c["id"].(string),
				Username: c["username"].(string),
				Email:    c["email"].(string),
				Exp:      int64(c["exp"].(float64)),
			}, jwtErrValidation
		}
		return nil, jwt.NewValidationError("invalid_token", jwt.ValidationErrorMalformed)
	}
	c := token.Claims.(jwt.MapClaims)
	return &modeltoken.TokenClaims{
		ID:       c["id"].(string),
		Username: c["username"].(string),
		Email:    c["email"].(string),
		Exp:      int64(c["exp"].(float64)),
	}, nil
}

type ChangePasswordRequest struct {
	Id          int    `param:"id" validate:"required"`
	OldPassword string `json:"old_password" form:"old_password" validate:"required"`
	NewPassword string `json:"new_password" form:"new_password" validate:"required"`
}

type ResetPasswordRequest struct {
	Id int `param:"id" validate:"required"`
}
