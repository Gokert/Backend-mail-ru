package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type contextKey string

const UserIDKey contextKey = "userId"

type Core interface {
	GetUserId(ctx context.Context, sid string) (uint64, error)
}

func AuthCheck(next http.Handler, core Core, lg *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("session_id")
		if errors.Is(err, http.ErrNoCookie) {
			next.ServeHTTP(w, r)
			return
		}

		userId, err := core.GetUserId(r.Context(), session.Value)
		if err != nil {
			lg.Error("auth check error", "err", err.Error())
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), UserIDKey, userId))

		next.ServeHTTP(w, r)
	})
}
