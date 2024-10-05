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
	Username string `json:"username"`
	Email    string `json:"email"`
	Exp      int64  `json:"exp"`

	jwt.RegisteredClaims
}

func (c TokenClaims) AuthContext() (*abstraction.AuthContext, error) {
	var (
		id       int
		username string
		email    string
		err      error

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

	destructUsername := c.Username
	if destructUsername == "" {
		return nil, errors.New("invalid_token")
	}
	if username, err = encoding.Decode(fmt.Sprintf("%v", destructUsername)); err != nil {
		return nil, errors.New("invalid_token")
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
		Username: username,
		Email:    email,
	}, nil
}
