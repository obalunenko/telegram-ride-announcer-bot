package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
	tgbotapi "github.com/mymmrac/telego"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/models"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/ops"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/sessions"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/states"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/trips"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/repository/users"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram"
	"github.com/obalunenko/telegram-ride-announcer-bot/internal/telegram/telegramtest"
)

const (
	testUserID  int64 = 42
	otherUserID int64 = 43
	testChatID  int64 = 4242
	otherChatID int64 = 4343

	msgNotFound       = "Command not found. Use /help command to see all available commands."
	msgNotImplemented = "Not implemented yet. Use /help command to see all available commands."
	msgFarewell       = "I'm going to sleep. Bye, John!"
)

var errTest = errors.New("test error")

// testRepos are in-memory repositories. A test replaces any of them to inject failures.
type testRepos struct {
	users    users.Repository
	sessions sessions.Repository
	states   states.Repository
	trips    trips.Repository
}

func newTestRepos() *testRepos {
	return &testRepos{
		users:    users.NewInMemory(),
		sessions: sessions.NewInMemory(),
		states:   states.NewInMemory(),
		trips:    trips.NewInMemory(),
	}
}

func (r *testRepos) UsersRepository() users.Repository {
	return r.users
}

func (r *testRepos) SessionsRepository() sessions.Repository {
	return r.sessions
}

func (r *testRepos) StatesRepository() states.Repository {
	return r.states
}

func (r *testRepos) TripsRepository() trips.Repository {
	return r.trips
}

// failingUsers fails the method named in it.
type failingUsers struct {
	users.Repository

	method string
}

func (f failingUsers) Create(ctx context.Context, user *users.User) error {
	if f.method == "Create" {
		return errTest
	}

	return f.Repository.Create(ctx, user)
}

func (f failingUsers) GetBuID(ctx context.Context, id int64) (*users.User, error) {
	if f.method == "GetBuID" {
		return nil, errTest
	}

	return f.Repository.GetBuID(ctx, id)
}

func (f failingUsers) Update(ctx context.Context, user *users.User) error {
	if f.method == "Update" {
		return errTest
	}

	return f.Repository.Update(ctx, user)
}

// failingSessions fails the method named in it.
type failingSessions struct {
	sessions.Repository

	method string
}

func (f failingSessions) CreateSession(ctx context.Context, userID, chatID int64, stateID *uuid.UUID) error {
	if f.method == "CreateSession" {
		return errTest
	}

	return f.Repository.CreateSession(ctx, userID, chatID, stateID)
}

func (f failingSessions) GetSession(ctx context.Context, userID, chatID int64) (*sessions.Session, error) {
	if f.method == "GetSession" {
		return nil, errTest
	}

	return f.Repository.GetSession(ctx, userID, chatID)
}

// failingStates fails to update states, so sessions can't be saved.
type failingStates struct {
	states.Repository
}

func (failingStates) UpdateState(context.Context, *states.State) error {
	return errTest
}

// failingTrips fails the method named in it.
type failingTrips struct {
	trips.Repository

	method string
}

func (f failingTrips) CreateTrip(ctx context.Context, name, date, description string, createdBy int64) (*trips.Trip, error) {
	if f.method == "CreateTrip" {
		return nil, errTest
	}

	return f.Repository.CreateTrip(ctx, name, date, description, createdBy)
}

func (f failingTrips) UpdateTrip(ctx context.Context, id uuid.UUID, params trips.UpdateTripParams) error {
	if f.method == "UpdateTrip" {
		return errTest
	}

	return f.Repository.UpdateTrip(ctx, id, params)
}

func (f failingTrips) DeleteTrip(ctx context.Context, id uuid.UUID) error {
	if f.method == "DeleteTrip" {
		return errTest
	}

	return f.Repository.DeleteTrip(ctx, id)
}

func (f failingTrips) ListTripsByUser(ctx context.Context, userID int64) ([]*trips.Trip, error) {
	if f.method == "ListTripsByUser" {
		return nil, errTest
	}

	return f.Repository.ListTripsByUser(ctx, userID)
}

// failingRenderer fails to render any template.
type failingRenderer struct{}

func (failingRenderer) Help(renderer.HelpParams) (string, error) {
	return "", errTest
}

func (failingRenderer) Welcome(renderer.WelcomeParams) (string, error) {
	return "", errTest
}

func (failingRenderer) Trip(renderer.TripParams) (string, error) {
	return "", errTest
}

// newTestService returns a Service that talks to a fake Bot API server.
func newTestService(t *testing.T, repos *testRepos) (*Service, *telegramtest.Server) {
	t.Helper()

	api := telegramtest.NewServer(t)

	bot, err := telegram.NewBot(context.Background(), telegramtest.Token,
		telegram.WithClientOptions(api.ClientOptions()...))
	require.NoError(t, err)

	// Skip the calls made by telegram.NewBot: getMe, getMyDescription, getMyCommands.
	for range 3 {
		api.NextRequest(t)
	}

	svc, err := New(bot, repos)
	require.NoError(t, err)

	return svc, api
}

// textUpdate returns an update with a text message sent to the test chat.
func textUpdate(from int64, text string) tgbotapi.Update {
	return chatUpdate(from, testChatID, text)
}

// chatUpdate returns an update with a text message sent to the chat.
func chatUpdate(from, chatID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From: &tgbotapi.User{
				ID:        from,
				FirstName: "John",
				Username:  "john",
			},
			Chat: tgbotapi.Chat{
				ID:   chatID,
				Type: tgbotapi.ChatTypePrivate,
			},
			Text: text,
		},
	}
}

// nextMessage returns the text of the next message the bot sends to the test chat.
func nextMessage(t *testing.T, api *telegramtest.Server) string {
	t.Helper()

	req := api.NextRequest(t)
	require.Equal(t, "sendMessage", req.Method)
	assert.EqualValues(t, testChatID, req.Params["chat_id"])

	text, ok := req.Params["text"].(string)
	require.True(t, ok, "message text is not a string: %v", req.Params["text"])

	return text
}

// replyTo sends a message from the test user and returns the text of the bot reply.
func replyTo(t *testing.T, api *telegramtest.Server, text string) string {
	t.Helper()

	api.SendUpdate(textUpdate(testUserID, text))

	return nextMessage(t, api)
}

// assertState checks the saved session state of the test user. Handlers save the session
// before they reply, so the state is checked right after the reply, without waiting.
func assertState(t *testing.T, repos *testRepos, want models.State) {
	t.Helper()

	sess, err := ops.GetSession(context.Background(), repos, &models.User{ID: testUserID}, testChatID)
	require.NoError(t, err)
	assert.Equal(t, want, sess.UserState.State)
}

func TestNew(t *testing.T) {
	svc, err := New(nil, newTestRepos())
	require.EqualError(t, err, "bot is nil")
	assert.Nil(t, svc)
}

func TestService_PublishTrip(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)

	svc.Start(ctx)

	assert.Contains(t, replyTo(t, api, "/start"), "Hello, John! 👋 Welcome to test_bot.")
	assert.Contains(t, replyTo(t, api, "/help"), "Welcome to test_bot Help!")

	assert.Equal(t, "Please enter trip name", replyTo(t, api, "/newtrip"))
	assertState(t, repos, models.StateNewTripName)

	assert.Equal(t, `Your trip name "Morning ride". Please select date and time`, replyTo(t, api, "Morning ride"))
	assertState(t, repos, models.StateNewTripDate)

	assert.Equal(t, "Please enter trip description", replyTo(t, api, "today"))
	assertState(t, repos, models.StateNewTripDescription)

	confirm := replyTo(t, api, "Easy pace")
	assert.Contains(t, confirm, "Title: Morning ride\nDescription: Easy pace\nDate: today\nCreated By: @john")
	assert.True(t, strings.HasSuffix(confirm, "Please confirm"), confirm)
	assertState(t, repos, models.StateNewTripConfirm)

	assert.True(t, strings.HasPrefix(replyTo(t, api, "yes"), "Trip is published!"))

	pin := api.NextRequest(t)
	assert.Equal(t, "pinChatMessage", pin.Method)
	assert.EqualValues(t, testChatID, pin.Params["chat_id"])
	assertState(t, repos, models.StateStart)

	myTrips := replyTo(t, api, "/mytrips")
	assert.True(t, strings.HasPrefix(myTrips, "Your trips:"), myTrips)
	assert.Contains(t, myTrips, "Title: Morning ride")

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}

func TestService_CancelTrip(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)

	svc.Start(ctx)

	steps := []struct {
		text  string
		state models.State
	}{
		{text: "/newtrip", state: models.StateNewTripName},
		{text: "Evening ride", state: models.StateNewTripDate},
		{text: "tomorrow", state: models.StateNewTripDescription},
		{text: "Fast pace", state: models.StateNewTripConfirm},
	}

	for _, step := range steps {
		replyTo(t, api, step.text)
		assertState(t, repos, step.state)
	}

	assert.Equal(t, "Your trip is canceled. Thank you!", replyTo(t, api, "no"))
	assertState(t, repos, models.StateNewTrip)

	assert.Equal(t, "You have no trips yet. Use /newtrip command to create a new trip.", replyTo(t, api, "/mytrips"))

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}

func TestService_Commands(t *testing.T) {
	ctx := context.Background()

	svc, api := newTestService(t, newTestRepos())

	svc.Start(ctx)

	// The bot ignores its own messages and updates without a message: neither gets a reply.
	api.SendUpdate(textUpdate(telegramtest.BotID, "/help"))
	api.SendUpdate(tgbotapi.Update{})

	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "unknown command",
			text: "/unknown",
			want: msgNotFound,
		},
		{
			name: "text outside of a trip",
			text: "hello",
			want: msgNotFound,
		},
		{
			name: "trips",
			text: "/" + CmdTrips,
			want: msgNotImplemented,
		},
		{
			name: "subscribe",
			text: "/" + CmdSubscribe,
			want: msgNotImplemented,
		},
		{
			name: "unsubscribe",
			text: "/" + CmdUnsubscribe,
			want: msgNotImplemented,
		},
		{
			name: "subscribed",
			text: "/" + CmdSubscribed,
			want: msgNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, replyTo(t, api, tt.text))
		})
	}

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}
