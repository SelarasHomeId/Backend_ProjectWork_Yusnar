package middleware

import (
	"errors"
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
			id        int
			role_id   int
			divisi_id int
			email     string
			jwtKey    = config.Get().JWT.SecretKey
		)
		authToken := c.Request().Header.Get("Authorization")
		if authToken == "" {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if !strings.Contains(authToken, "Bearer") {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
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
						return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
					}
					if _, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
						if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
							return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
						}
						if _, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
							return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
						}
					}
					response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "token_is_expired").SendError(c)
					return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "token_is_expired").SendError(c)
				}
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.ErrorBuilder(http.StatusUnauthorized, err, "error when claim token").SendError(c)
		}

		destructID := claims["id"]
		if destructID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructRoleID := claims["role_id"]
		if destructRoleID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
			if destructRoleID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructRoleID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructDivisiID := claims["divisi_id"]
		if destructDivisiID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
			if destructDivisiID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructDivisiID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructEmail := claims["email"]
		if destructEmail == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}

		cc := c.(*abstraction.Context)
		cc.Auth = &abstraction.AuthContext{
			ID:       id,
			RoleID:   role_id,
			DivisiID: divisi_id,
			Email:    email,
		}

		return next(cc)
	}
}

func Logout(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			id        int
			role_id   int
			divisi_id int
			email     string
			jwtKey    = config.Get().JWT.SecretKey
		)
		authToken := c.Request().Header.Get("Authorization")
		if authToken == "" {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if !strings.Contains(authToken, "Bearer") {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
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
					return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), errJWT.Error()).SendError(c)
				}
			} else {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.ErrorBuilder(http.StatusUnauthorized, err, "error when claim token").SendError(c)
		}

		destructID := claims["id"]
		if destructID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructRoleID := claims["role_id"]
		if destructRoleID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
			if destructRoleID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructRoleID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructDivisiID := claims["divisi_id"]
		if destructDivisiID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
			if destructDivisiID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructDivisiID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructEmail := claims["email"]
		if destructEmail == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}

		cc := c.(*abstraction.Context)
		cc.Auth = &abstraction.AuthContext{
			ID:       id,
			RoleID:   role_id,
			DivisiID: divisi_id,
			Email:    email,
		}

		return next(cc)
	}
}

func RefreshToken(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			id        int
			role_id   int
			divisi_id int
			email     string
			jwtKey    = config.Get().JWT.SecretKey
		)
		authToken := c.Request().Header.Get("Authorization")
		if authToken == "" {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if !strings.Contains(authToken, "Bearer") {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		tokenString := strings.Replace(authToken, "Bearer ", "", -1)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method :%v", token.Header["alg"])
			}
			return []byte(jwtKey), nil
		})

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.ErrorBuilder(http.StatusUnauthorized, err, "error when claim token").SendError(c)
		}

		destructID := claims["id"]
		if destructID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
			if destructID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if id, err = strconv.Atoi(fmt.Sprintf("%v", destructID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructRoleID := claims["role_id"]
		if destructRoleID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
			if destructRoleID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructRoleID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if role_id, err = strconv.Atoi(fmt.Sprintf("%v", destructRoleID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructDivisiID := claims["divisi_id"]
		if destructDivisiID == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
			if destructDivisiID, err = aescrypt.DecryptAES(fmt.Sprintf("%v", destructDivisiID), jwtKey); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
			if divisi_id, err = strconv.Atoi(fmt.Sprintf("%v", destructDivisiID)); err != nil {
				return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
			}
		}

		destructEmail := claims["email"]
		if destructEmail == nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}
		if email, err = encoding.Decode(fmt.Sprintf("%v", destructEmail)); err != nil {
			return response.ErrorBuilder(http.StatusUnauthorized, errors.New("unauthorized"), "invalid_token").SendError(c)
		}

		cc := c.(*abstraction.Context)
		cc.Auth = &abstraction.AuthContext{
			ID:       id,
			RoleID:   role_id,
			DivisiID: divisi_id,
			Email:    email,
		}

		return next(cc)
	}
}
