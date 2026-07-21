package middleware

import (
	"strings"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
	"github.com/ihsanuta/task-management-api/pkg/response"
	"github.com/labstack/echo/v4"
)

type authCtxKey string

const (
	UserIDKey authCtxKey = "user_id"
	TeamIDKey authCtxKey = "team_id"
	EmailKey  authCtxKey = "email"
)

func Auth(jwtManager *jwtutil.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				return response.Error(c, apperror.ErrInvalidToken)
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwtManager.Parse(tokenStr)
			if err != nil {
				return response.Error(c, apperror.ErrInvalidToken)
			}

			c.Set(string(UserIDKey), claims.UserID)
			c.Set(string(TeamIDKey), claims.TeamID)
			c.Set(string(EmailKey), claims.Email)

			return next(c)
		}
	}
}

func UserID(c echo.Context) string {
	v, _ := c.Get(string(UserIDKey)).(string)
	return v
}

func TeamID(c echo.Context) string {
	v, _ := c.Get(string(TeamIDKey)).(string)
	return v
}
