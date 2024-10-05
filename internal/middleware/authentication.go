package middleware

import (
	"fmt"
	"net/http"
	"selarashomeid/internal/abstraction"
	"selarashomeid/internal/config"
	"selarashomeid/pkg/util/aescrypt"
	"selarashomeid/pkg/util/encoding"
	"selarashomeid/pkg/util/response"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func Authentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			id       int
			username string
			email    string
			jwtKey   = config.Get().JWT.SecretKey
		)
		authToken := c.Request().Header.Get("Authorization")
		if authToken == "" {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if !strings.Contains(authToken, "Bearer") {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		tokenString := strings.Replace(authToken, "Bearer ", "", -1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method :%v", token.Header["alg"])
			}
			return []byte(jwtKey), nil
		})
		if token == nil || !token.Valid || err != nil {
			if errJWT, ok := err.(*jwt.ValidationError); ok {
				if errJWT.Errors == jwt.ValidationErrorExpired {
					destructID := token.Claims.(jwt.MapClaims)["id"]
					if destructID == nil {
						return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
					}
					if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
						if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
							return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
						}
						if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
							return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
						}
					}
					return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "token_is_expired").Send(c)
				}
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.ErrorBuilder(&response.ErrorConstant.Unauthorized, err).Send(c)
		}

		destructID := claims["id"]
		if destructID == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
			if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
		}
		destructUsername := claims["username"]
		if destructUsername == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if username, err = encoding.Decode(fmt.Sprintf("%v", destructUsername)); err != nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}

		destructEmail := claims["email"]
		if destructEmail == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}

		cc := c.(*abstraction.Context)
		cc.Auth = &abstraction.AuthContext{
			ID:       id,
			Username: username,
			Email:    email,
		}

		return next(cc)
	}
}

func Logout(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			id       int
			username string
			email    string
			jwtKey   = config.Get().JWT.SecretKey
		)
		authToken := c.Request().Header.Get("Authorization")
		if authToken == "" {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if !strings.Contains(authToken, "Bearer") {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		tokenString := strings.Replace(authToken, "Bearer ", "", -1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method :%v", token.Header["alg"])
			}
			return []byte(jwtKey), nil
		})
		if token == nil || !token.Valid || err != nil {
			if errJWT, ok := err.(*jwt.ValidationError); ok {
				if errJWT.Errors != jwt.ValidationErrorExpired {
					return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, errJWT.Error()).Send(c)
				}
			} else {
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.ErrorBuilder(&response.ErrorConstant.Unauthorized, err).Send(c)
		}

		destructID := claims["id"]
		if destructID == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
			if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
				return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
			}
		}
		destructUsername := claims["username"]
		if destructUsername == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if username, err = encoding.Decode(fmt.Sprintf("%v", destructUsername)); err != nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		destructEmail := claims["email"]
		if destructEmail == nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}
		if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
			return response.CustomErrorBuilder(http.StatusUnauthorized, response.E_UNAUTHORIZED, "invalid_token").Send(c)
		}

		cc := c.(*abstraction.Context)
		cc.Auth = &abstraction.AuthContext{
			ID:       id,
			Username: username,
			Email:    email,
		}

		return next(cc)
	}
}
