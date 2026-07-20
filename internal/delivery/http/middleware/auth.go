package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ihsanuta/task-management-api/pkg/apperror"
	"github.com/ihsanuta/task-management-api/pkg/jwtutil"
	"github.com/ihsanuta/task-management-api/pkg/response"
)

type authCtxKey string

const (
	UserIDKey authCtxKey = "user_id"
	TeamIDKey authCtxKey = "team_id"
	EmailKey  authCtxKey = "email"
)

// Auth validates the Bearer JWT on every protected route and injects the
// authenticated user's id/team/email into the request context.
func Auth(jwtManager *jwtutil.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Error(w, apperror.ErrInvalidToken)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := jwtManager.Parse(tokenStr)
			if err != nil {
				response.Error(w, apperror.ErrInvalidToken)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, TeamIDKey, claims.TeamID)
			ctx = context.WithValue(ctx, EmailKey, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

func TeamID(ctx context.Context) string {
	v, _ := ctx.Value(TeamIDKey).(string)
	return v
}
