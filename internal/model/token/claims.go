package token

import (
	"errors"
	"fmt"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/config"
	"selarashomeid/pkg/util/aescrypt"
	"selarashomeid/pkg/util/encoding"
	"strconv"

	"github.com/golang-jwt/jwt/v4"
)

type TokenClaims struct {
	ID       string `json:"id"`
	RoleID   string `json:"role_id"`
	DivisiID string `json:"divisi_id"`
	Email    string `json:"email"`
	Exp      int64  `json:"exp"`

	jwt.RegisteredClaims
}

func (c TokenClaims) AuthContext() (*abstraction.AuthContext, error) {
	var (
		id        int
		role_id   int
		divisi_id int
		email     string
		err       error

		encryptionKey = config.Get().JWT.SecretKey
	)

	destructID := c.ID
	if destructID == "" {
		return nil, errors.New("invalid_token")
	}
	if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
		if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), encryptionKey); err != nil {
			return nil, errors.New("invalid_token")
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			return nil, errors.New("invalid_token")
		}
	}

	destructRoleID := c.RoleID
	if destructRoleID == "" {
		return nil, errors.New("invalid_token")
	}
	if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
		if destructRoleID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructRoleID), encryptionKey); err != nil {
			return nil, errors.New("invalid_token")
		}
		if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
			return nil, errors.New("invalid_token")
		}
	}

	destructDivisiID := c.DivisiID
	if destructDivisiID == "" {
		return nil, errors.New("invalid_token")
	}
	if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
		if destructDivisiID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructDivisiID), encryptionKey); err != nil {
			return nil, errors.New("invalid_token")
		}
		if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
			return nil, errors.New("invalid_token")
		}
	}

	destructEmail := c.Email
	if destructEmail == "" {
		return nil, errors.New("invalid_token")
	}
	if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
		return nil, errors.New("invalid_token")
	}

	return &abstraction.AuthContext{
		ID:       id,
		RoleID:   role_id,
		DivisiID: divisi_id,
		Email:    email,
	}, nil
}
