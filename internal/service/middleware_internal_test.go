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
		name   string
		update tgbotapi.Update
		setup  func(t *testing.T, repos *testRepos)
		// wantState is the state of the session passed to the next handler; 0 means the chain is stopped.
		wantState models.State
		// wantChat is the chat of the session passed to the next handler; 0 means the test chat.
		wantChat int64
		// wantError means the chain is stopped with an error reply to the user.
		wantError bool
	}{
		{
			name:      "new user",
			update:    textUpdate(testUserID, "hello"),
			wantState: models.StateStart,
		},
		{
			name:   "existing session",
			update: textUpdate(testUserID, "hello"),
			setup: func(t *testing.T, repos *testRepos) {
				sess := newSession(t, repos, testUserID, models.StateNewTripName)
				require.NoError(t, ops.UpdateSession(context.Background(), repos, sess))
			},
			wantState: models.StateNewTripName,
		},
		{
			name:   "same user in another chat",
			update: chatUpdate(testUserID, otherChatID, "hello"),
			setup: func(t *testing.T, repos *testRepos) {
				sess := newSession(t, repos, testUserID, models.StateNewTripName)
				require.NoError(t, ops.UpdateSession(context.Background(), repos, sess))
			},
			wantState: models.StateStart,
			wantChat:  otherChatID,
		},
		{
			name:   "changed profile",
			update: textUpdate(testUserID, "hello"),
			setup: func(t *testing.T, repos *testRepos) {
				createUser(t, repos, "old_john")
			},
			wantState: models.StateStart,
		},
		{
			name:   "update user fails",
			update: textUpdate(testUserID, "hello"),
			setup: func(t *testing.T, repos *testRepos) {
				createUser(t, repos, "old_john")
				repos.users = failingUsers{Repository: repos.users, method: "Update"}
			},
			wantError: true,
		},
		{
			name:   "message from the bot",
			update: textUpdate(telegramtest.BotID, "hello"),
		},
		{
			name:   "update without a message",
			update: tgbotapi.Update{},
		},
		{
			name: "message without a sender",
			update: tgbotapi.Update{
				Message: &tgbotapi.Message{Chat: tgbotapi.Chat{ID: testChatID}, Text: "hello"},
			},
		},
		{
			name:   "get user fails",
			update: textUpdate(testUserID, "hello"),
			setup: func(_ *testing.T, repos *testRepos) {
				repos.users = failingUsers{Repository: repos.users, method: "GetBuID"}
			},
			wantError: true,
		},
		{
			name:   "create user fails",
			update: textUpdate(testUserID, "hello"),
			setup: func(_ *testing.T, repos *testRepos) {
				repos.users = failingUsers{Repository: repos.users, method: "Create"}
			},
			wantError: true,
		},
		{
			name:   "get session fails",
			update: textUpdate(testUserID, "hello"),
			setup: func(_ *testing.T, repos *testRepos) {
				repos.sessions = failingSessions{Repository: repos.sessions, method: "GetSession"}
			},
			wantError: true,
		},
		{
			name:   "create session fails",
			update: textUpdate(testUserID, "hello"),
			setup: func(_ *testing.T, repos *testRepos) {
				repos.sessions = failingSessions{Repository: repos.sessions, method: "CreateSession"}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := newTestRepos()
			svc, api := newTestService(t, repos)

			if tt.setup != nil {
				tt.setup(t, repos)
			}

			next := handleUpdate(t, svc, svc.setSessionMiddleware(), tt.update)
			if tt.wantState == 0 {
				assert.Nil(t, next)

				if tt.wantError {
					assert.Equal(t, msgSomethingWentWrong, nextMessage(t, api))
				}

				api.AssertNoRequest(t)

				return
			}

			require.NotNil(t, next)

			wantChat := tt.wantChat
			if wantChat == 0 {
				wantChat = testChatID
			}

			sess := sessionFromContext(next.Context())
			require.NotNil(t, sess)
			assert.Equal(t, &models.User{ID: testUserID, Username: "john", Firstname: "John"}, sess.User)
			assert.Equal(t, wantChat, sess.ChatID)
			assert.Equal(t, tt.wantState, sess.UserState.State)
		})
	}
}

// createUser stores the test user with the username.
func createUser(t *testing.T, repos *testRepos, username string) {
	t.Helper()

	_, err := ops.CreateUser(context.Background(), repos, ops.CreateUserParams{
		UserID:    testUserID,
		Username:  username,
		Firstname: "John",
	})
	require.NoError(t, err)
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
