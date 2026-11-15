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

// newSession creates the user with a session in the state.
func newSession(t *testing.T, repos *testRepos, userID int64, state models.State) *models.Session {
	t.Helper()

	ctx := context.Background()

	user, err := ops.CreateUser(ctx, repos, ops.CreateUserParams{
		UserID:    userID,
		Username:  "john",
		Firstname: "John",
		Lastname:  "",
	})
	require.NoError(t, err)

	sess, err := ops.CreateSession(ctx, repos, ops.CreateSessionParams{
		User:   user,
		ChatID: testChatID,
	})
	require.NoError(t, err)

	sess.UserState.State = state

	return sess
}

// newTrip creates a trip of the user.
func newTrip(t *testing.T, repos *testRepos, createdBy int64) *models.Trip {
	t.Helper()

	trip, err := ops.CreateTrip(context.Background(), repos, ops.CreateTripParams{
		Name:        "Morning ride",
		Date:        testNow,
		Description: "Easy pace",
		CreatedBy:   createdBy,
	})
	require.NoError(t, err)

	return trip
}

// newPublishedTrip creates a published trip of the user.
func newPublishedTrip(t *testing.T, repos *testRepos, createdBy int64) *models.Trip {
	t.Helper()

	trip := newTrip(t, repos, createdBy)
	completed := true

	trip, err := ops.UpdateTrip(context.Background(), repos, trip.ID, ops.UpdateTripParams{Completed: &completed})
	require.NoError(t, err)

	return trip
}

// sessionUpdate returns a text message update that carries the session in its context.
func sessionUpdate(sess *models.Session, text string) tgbotapi.Update {
	return textUpdate(testUserID, text).WithContext(contextWithSession(context.Background(), sess))
}

func TestHandlers_NoSession(t *testing.T) {
	ctx := context.Background()

	svc, api := newTestService(t, newTestRepos())

	tests := []struct {
		name    string
		handler th.Handler
	}{
		{name: "start", handler: svc.startHandler(ctx)},
		{name: "help", handler: svc.helpHandler()},
		{name: "text", handler: svc.textHandler()},
		{name: "new trip", handler: svc.newTripHandler()},
		{name: "my trips", handler: svc.myTripsHandler()},
		{name: "not found", handler: svc.notFoundHandler(ctx)},
		{name: "not implemented", handler: svc.notImplementedHandler(CmdTrips)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.handler(nil, textUpdate(testUserID, "hello")))
			api.AssertNoRequest(t)
		})
	}

	require.EqualError(t, svc.createTrip(ctx, textUpdate(testUserID, "hello")), "session is nil")
	api.AssertNoRequest(t)
}

func TestHandlers_Failure(t *testing.T) {
	tests := []struct {
		name    string
		handler func(svc *Service) th.Handler
		// setup prepares the failure; it can set a draft trip on the session.
		setup func(t *testing.T, svc *Service, repos *testRepos, sess *models.Session)
		// wantError means the handler replies with the error message.
		wantError bool
	}{
		{
			name: "start: save session",
			handler: func(svc *Service) th.Handler {
				return svc.startHandler(context.Background())
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantError: true,
		},
		{
			name: "start: discard draft",
			handler: func(svc *Service) th.Handler {
				return svc.startHandler(context.Background())
			},
			setup: func(t *testing.T, _ *Service, repos *testRepos, sess *models.Session) {
				sess.UserState.Trip = newTrip(t, repos, testUserID)
				repos.trips = failingTrips{Repository: repos.trips, method: "DeleteTrip"}
			},
			wantError: true,
		},
		{
			name: "start: render welcome",
			handler: func(svc *Service) th.Handler {
				return svc.startHandler(context.Background())
			},
			setup: func(_ *testing.T, svc *Service, _ *testRepos, _ *models.Session) {
				svc.templates = failingRenderer{}
			},
			wantError: true,
		},
		{
			name: "help: save session",
			handler: func(svc *Service) th.Handler {
				return svc.helpHandler()
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantError: true,
		},
		{
			name: "help: render help",
			handler: func(svc *Service) th.Handler {
				return svc.helpHandler()
			},
			setup: func(_ *testing.T, svc *Service, _ *testRepos, _ *models.Session) {
				svc.templates = failingRenderer{}
			},
			wantError: true,
		},
		{
			name: "my trips: list trips",
			handler: func(svc *Service) th.Handler {
				return svc.myTripsHandler()
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.trips = failingTrips{Repository: repos.trips, method: "ListTripsByUser"}
			},
		},
		{
			name: "my trips: render trip",
			handler: func(svc *Service) th.Handler {
				return svc.myTripsHandler()
			},
			setup: func(t *testing.T, svc *Service, repos *testRepos, _ *models.Session) {
				newPublishedTrip(t, repos, testUserID)

				svc.templates = failingRenderer{}
			},
		},
		{
			name: "new trip: save session",
			handler: func(svc *Service) th.Handler {
				return svc.newTripHandler()
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantError: true,
		},
		{
			name: "new trip: create trip",
			handler: func(svc *Service) th.Handler {
				return svc.newTripHandler()
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.trips = failingTrips{Repository: repos.trips, method: "CreateTrip"}
			},
			wantError: true,
		},
		{
			name: "trip answer: update trip",
			handler: func(svc *Service) th.Handler {
				return svc.tripAnswerHandler()
			},
			setup: func(t *testing.T, _ *Service, repos *testRepos, sess *models.Session) {
				sess.UserState.State = models.StateNewTripName
				sess.UserState.Trip = newTrip(t, repos, testUserID)
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantError: true,
		},
		{
			name: "cancel: save session",
			handler: func(svc *Service) th.Handler {
				return svc.cancelHandler()
			},
			setup: func(_ *testing.T, _ *Service, repos *testRepos, _ *models.Session) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := newTestRepos()
			svc, api := newTestService(t, repos)
			sess := newSession(t, repos, testUserID, models.StateStart)

			tt.setup(t, svc, repos, sess)

			// Handlers log failures and return no error to telego.
			require.NoError(t, tt.handler(svc)(nil, sessionUpdate(sess, "hello")))

			if tt.wantError {
				assert.Equal(t, msgSomethingWentWrong, nextMessage(t, api))
			}

			api.AssertNoRequest(t)
		})
	}
}

func TestTextHandler_UnknownState(t *testing.T) {
	repos := newTestRepos()
	svc, api := newTestService(t, repos)
	sess := newSession(t, repos, testUserID, models.State(0))

	require.NoError(t, svc.textHandler()(nil, sessionUpdate(sess, "hello")))
	assert.Equal(t, msgNotFound, nextMessage(t, api))
}

func TestService_CreateTrip(t *testing.T) {
	tests := []struct {
		name  string
		state models.State
		text  string
		// tripBy creates the session trip by the user; 0 leaves the session without a trip.
		tripBy  int64
		setup   func(svc *Service, repos *testRepos, api *telegramtest.Server)
		wantErr string
	}{
		{
			name:  "start: create trip",
			state: models.StateNewTrip,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "CreateTrip"}
			},
			wantErr: errTest.Error(),
		},
		{
			name:   "name: update trip",
			state:  models.StateNewTripName,
			text:   "Morning ride",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantErr: "failed to update trip",
		},
		{
			name:  "start: save session",
			state: models.StateNewTrip,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantErr: "failed to update session",
		},
		{
			name:   "name: send reply",
			state:  models.StateNewTripName,
			text:   "Morning ride",
			tripBy: testUserID,
			setup: func(_ *Service, _ *testRepos, api *telegramtest.Server) {
				api.Fail("sendMessage")
			},
			wantErr: "failed to send message",
		},
		{
			name:   "date: update trip",
			state:  models.StateNewTripDate,
			text:   "today",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantErr: "failed to update trip",
		},
		{
			name:   "time: update trip",
			state:  models.StateNewTripTime,
			text:   "09:30",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantErr: "failed to update trip",
		},
		{
			name:   "time: send reply",
			state:  models.StateNewTripTime,
			text:   "09:30",
			tripBy: testUserID,
			setup: func(_ *Service, _ *testRepos, api *telegramtest.Server) {
				api.Fail("sendMessage")
			},
			wantErr: "failed to send message",
		},
		{
			name:   "description: update trip",
			state:  models.StateNewTripDescription,
			text:   "Easy pace",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantErr: "failed to update trip",
		},
		{
			name:    "description: not the trip creator",
			state:   models.StateNewTripDescription,
			text:    "Easy pace",
			tripBy:  otherUserID,
			wantErr: "is not the creator of the trip",
		},
		{
			name:   "description: render trip",
			state:  models.StateNewTripDescription,
			text:   "Easy pace",
			tripBy: testUserID,
			setup: func(svc *Service, _ *testRepos, _ *telegramtest.Server) {
				svc.templates = failingRenderer{}
			},
			wantErr: "failed to render trip",
		},
		{
			name:   "description: send confirmation",
			state:  models.StateNewTripDescription,
			text:   "Easy pace",
			tripBy: testUserID,
			setup: func(_ *Service, _ *testRepos, api *telegramtest.Server) {
				api.Fail("sendMessage")
			},
			wantErr: "failed to send message",
		},
		{
			name:   "cancel: delete trip",
			state:  models.StateNewTripConfirm,
			text:   "no",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "DeleteTrip"}
			},
			wantErr: "failed to delete trip",
		},
		{
			name:   "publish: update trip",
			state:  models.StateNewTripConfirm,
			text:   "yes",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.trips = failingTrips{Repository: repos.trips, method: "UpdateTrip"}
			},
			wantErr: "failed to update trip",
		},
		{
			name:    "publish: not the trip creator",
			state:   models.StateNewTripConfirm,
			text:    "yes",
			tripBy:  otherUserID,
			wantErr: "is not the creator of the trip",
		},
		{
			name:   "publish: render trip",
			state:  models.StateNewTripConfirm,
			text:   "yes",
			tripBy: testUserID,
			setup: func(svc *Service, _ *testRepos, _ *telegramtest.Server) {
				svc.templates = failingRenderer{}
			},
			wantErr: "failed to render trip",
		},
		{
			name:   "publish: save session",
			state:  models.StateNewTripConfirm,
			text:   "yes",
			tripBy: testUserID,
			setup: func(_ *Service, repos *testRepos, _ *telegramtest.Server) {
				repos.states = failingStates{Repository: repos.states}
			},
			wantErr: "failed to update session",
		},
		{
			name:   "publish: send announcement",
			state:  models.StateNewTripConfirm,
			text:   "yes",
			tripBy: testUserID,
			setup: func(_ *Service, _ *testRepos, api *telegramtest.Server) {
				api.Fail("sendMessage")
			},
			wantErr: "failed to send message",
		},
		{
			// A failed pin does not fail the step: the trip is published already.
			name:   "publish: pin announcement",
			state:  models.StateNewTripConfirm,
			text:   "yes",
			tripBy: testUserID,
			setup: func(_ *Service, _ *testRepos, api *telegramtest.Server) {
				api.Fail("pinChatMessage")
			},
			wantErr: "",
		},
		{
			name:    "unexpected state",
			state:   models.StateStart,
			text:    "yes",
			tripBy:  testUserID,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := newTestRepos()
			svc, api := newTestService(t, repos)
			sess := newSession(t, repos, testUserID, tt.state)

			if tt.tripBy != 0 {
				if tt.tripBy != testUserID {
					newSession(t, repos, tt.tripBy, models.StateStart)
				}

				sess.UserState.Trip = newTrip(t, repos, tt.tripBy)
			}

			if tt.setup != nil {
				tt.setup(svc, repos, api)
			}

			update := sessionUpdate(sess, tt.text)

			err := svc.createTrip(update.Context(), update)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
