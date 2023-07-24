package main

import "context"

type sessionKey struct{}

func sessionFromContext(ctx context.Context) *session {
	sess, ok := ctx.Value(sessionKey{}).(*session)
	if !ok {
		return nil
	}

	return sess
}

func contextWithSession(ctx context.Context, sess *session) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}
