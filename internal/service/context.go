package service

import (
	"context"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
)

type sessionKey struct{}

// sessionFromContext returns a session from the context.
func sessionFromContext(ctx context.Context) *models.Session {
	sess, ok := ctx.Value(sessionKey{}).(*models.Session)
	if !ok || sess == nil {
		return nil
	}

	return sess
}

// contextWithSession returns a new context with the session.
func contextWithSession(ctx context.Context, sess *models.Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}
