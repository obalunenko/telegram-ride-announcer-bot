package service

import (
	"context"
	"testing"

	tgbotapi "github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram/telegramtest"
)

// handleUpdate runs the update through the middleware and returns the update it passes
// to the next handler, or nil if the middleware stops the chain.
func handleUpdate(t *testing.T, svc *Service, middleware th.Handler, update tgbotapi.Update) *tgbotapi.Update {
	t.Helper()

	var (
		group th.HandlerGroup
		next  *tgbotapi.Update
	)

	group.Use(middleware)
	group.Handle(func(_ *th.Context, update tgbotapi.Update) error {
		next = &update

		return nil
	})

	require.NoError(t, group.HandleUpdate(context.Background(), svc.bot.Client(), update))

	return next
}

func TestSetContextMiddleware(t *testing.T) {
	type ctxKey struct{}

	svc, _ := newTestService(t, newTestRepos())
	ctx := context.WithValue(context.Background(), ctxKey{}, "value")

	next := handleUpdate(t, svc, svc.setContextMiddleware(ctx), textUpdate(testUserID, "hello"))
	require.NotNil(t, next)
	assert.Equal(t, "value", next.Context().Value(ctxKey{}))
}

func TestLoggerMiddleware(t *testing.T) {
	repos := newTestRepos()
	svc, _ := newTestService(t, repos)
	sess := newSession(t, repos, testUserID, models.StateStart)

	tests := []struct {
		name   string
		update tgbotapi.Update
	}{
		{
			name:   "with session",
			update: sessionUpdate(sess, "hello"),
		},
		{
			name:   "without session",
			update: textUpdate(testUserID, "hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, handleUpdate(t, svc, svc.loggerMiddleware(), tt.update))
		})
	}
}

func TestSetSessionMiddleware(t *testing.T) {
	tests := []struct {
		name  string
		from  int64
		setup func(t *testing.T, repos *testRepos)
		// wantState is the state of the session passed to the next handler; 0 means the chain is stopped.
		wantState models.State
	}{
		{
			name:      "new user",
			from:      testUserID,
			wantState: models.StateStart,
		},
		{
			name: "existing session",
			from: testUserID,
			setup: func(t *testing.T, repos *testRepos) {
				sess := newSession(t, repos, testUserID, models.StateNewTripName)
				require.NoError(t, ops.UpdateSession(context.Background(), repos, sess))
			},
			wantState: models.StateNewTripName,
		},
		{
			name: "message from the bot",
			from: telegramtest.BotID,
		},
		{
			name: "create user fails",
			from: testUserID,
			setup: func(_ *testing.T, repos *testRepos) {
				repos.users = failingUsers{Repository: repos.users}
			},
		},
		{
			name: "get session fails",
			from: testUserID,
			setup: func(_ *testing.T, repos *testRepos) {
				repos.sessions = failingSessions{Repository: repos.sessions, method: "GetSessionByUserID"}
			},
		},
		{
			name: "create session fails",
			from: testUserID,
			setup: func(_ *testing.T, repos *testRepos) {
				repos.sessions = failingSessions{Repository: repos.sessions, method: "CreateSession"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := newTestRepos()
			svc, _ := newTestService(t, repos)

			if tt.setup != nil {
				tt.setup(t, repos)
			}

			next := handleUpdate(t, svc, svc.setSessionMiddleware(), textUpdate(tt.from, "hello"))
			if tt.wantState == 0 {
				assert.Nil(t, next)

				return
			}

			require.NotNil(t, next)

			sess := sessionFromContext(next.Context())
			require.NotNil(t, sess)
			assert.Equal(t, &models.User{ID: testUserID, Username: "john", Firstname: "John"}, sess.User)
			assert.Equal(t, testChatID, sess.ChatID)
			assert.Equal(t, tt.wantState, sess.UserState.State)
		})
	}
}

func TestPanicRecovery(t *testing.T) {
	svc, _ := newTestService(t, newTestRepos())

	var group th.HandlerGroup

	group.Use(svc.panicRecovery())
	group.Handle(func(*th.Context, tgbotapi.Update) error {
		panic("handler panic")
	})

	assert.NotPanics(t, func() {
		assert.NoError(t, group.HandleUpdate(context.Background(), svc.bot.Client(), textUpdate(testUserID, "hello")))
	})
}
