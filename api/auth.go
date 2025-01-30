package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"github.com/spf13/viper"
	"github.com/tifye/shigure/assert"
)

func requireAuthMiddleware(logger *log.Logger, config *viper.Viper) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.NoContent(http.StatusUnauthorized)
			}

			signingKey := config.GetString("JWT_Signing_Key")
			assert.AssertNotEmpty(signingKey)

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(signingKey), nil
			}, jwt.WithExpirationRequired())
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					return c.String(http.StatusUnauthorized, "token expired")
				}

				if errors.Is(err, jwt.ErrTokenMalformed) {
					return c.String(http.StatusBadRequest, "malformed token")
				}

				logger.Debug("token parse fail", "err", err)
				return c.NoContent(http.StatusBadRequest)
			}

			return next(c)
		}
	}
}

func handleGetToken(logger *log.Logger, config *viper.Viper) echo.HandlerFunc {
	return func(c echo.Context) error {
		secret := config.GetString("OTP_Secret")
		assert.AssertNotEmpty(secret)

		passcode := c.Request().Header.Get("Passcode")
		if passcode == "" {
			return c.NoContent(http.StatusBadRequest)
		}

		didPass := totp.Validate(passcode, secret)
		if !didPass {
			return c.NoContent(http.StatusUnauthorized)
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		})
		signingKey := config.GetString("JWT_Signing_Key")
		assert.AssertNotEmpty(signingKey)
		signed, err := token.SignedString([]byte(signingKey))
		if err != nil {
			logger.Error("jwt sign:", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.String(http.StatusOK, signed)
	}
}
