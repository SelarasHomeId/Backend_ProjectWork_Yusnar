package token

import (
	"selarashomeid/internal/config"

	"github.com/golang-jwt/jwt/v4"
)

type AuthToken struct {
	token *jwt.Token
}

func NewAuthToken(claims *TokenClaims) *AuthToken {
	return &AuthToken{token: jwt.NewWithClaims(jwt.SigningMethodHS256, claims)}
}

func (t *AuthToken) Token() (string, error) {
	signedString, err := t.token.SignedString([]byte(config.Get().JWT.SecretKey))
	if err != nil {
		return "", err
	}
	return signedString, nil
}
