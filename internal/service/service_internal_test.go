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

func (f failingSessions) ListSessions(ctx context.Context) ([]*sessions.Session, error) {
	if f.method == "ListSessions" {
		return nil, errTest
	}

	return f.Repository.ListSessions(ctx)
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

	// telegram.NewBot calls getMe, getMyDescription, getMyShortDescription, and getMyCommands.
	return newTestServiceWithAPI(t, repos, api, 4)
}

// newTestServiceWithAPI returns a Service with the bot options that talks to the api.
// It skips the startupCalls Bot API calls that telegram.NewBot makes.
func newTestServiceWithAPI(
	t *testing.T,
	repos *testRepos,
	api *telegramtest.Server,
	startupCalls int,
	opts ...telegram.BotOption,
) (*Service, *telegramtest.Server) {
	t.Helper()

	opts = append([]telegram.BotOption{telegram.WithClientOptions(api.ClientOptions()...)}, opts...)

	bot, err := telegram.NewBot(context.Background(), telegramtest.Token, opts...)
	require.NoError(t, err)

	for range startupCalls {
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

func TestService_Shutdown_ListSessionsFails(t *testing.T) {
	repos := newTestRepos()
	svc, api := newTestService(t, repos)

	newSession(t, repos, testUserID, models.StateStart)

	repos.sessions = failingSessions{Repository: repos.sessions, method: "ListSessions"}

	var stopped bool

	svc.stopFns = append(svc.stopFns, func(context.Context) {
		stopped = true
	})

	svc.Shutdown(context.Background())

	assert.True(t, stopped, "the bot must stop even when it fails to say goodbye")
	api.AssertNoRequest(t)
}

func TestService_PublishTrip(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)

	svc.Start(ctx)

	assert.Contains(t, replyTo(t, api, "/start"), "Hello, John! 👋 Welcome to @test_bot.")
	assert.Contains(t, replyTo(t, api, "/help"), "Welcome to @test_bot Help!")

	assert.Equal(t, msgEnterTripName, replyTo(t, api, "/newtrip"))
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

	published, err := ops.ListTripsByUser(ctx, repos, &models.User{ID: testUserID})
	require.NoError(t, err)
	require.Len(t, published, 1)
	assert.True(t, published[0].Completed, "a published trip is completed")

	myTrips := replyTo(t, api, "/mytrips")
	assert.True(t, strings.HasPrefix(myTrips, "Your trips:"), myTrips)
	assert.Contains(t, myTrips, "Title: Morning ride")

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}

// toConfirmation runs the trip dialog up to the confirmation step.
func toConfirmation(t *testing.T, api *telegramtest.Server, repos *testRepos) {
	t.Helper()

	replyTo(t, api, "/newtrip")
	replyTo(t, api, "Morning ride")
	replyTo(t, api, "today")
	replyTo(t, api, "Easy pace")
	assertState(t, repos, models.StateNewTripConfirm)
}

func TestService_NewTripRestarts(t *testing.T) {
	ctx := context.Background()

	repos := newTestRepos()
	svc, api := newTestService(t, repos)

	svc.Start(ctx)

	replyTo(t, api, "/newtrip")
	replyTo(t, api, "Morning ride")
	assertState(t, repos, models.StateNewTripDate)

	assert.Equal(t, msgDraftDiscarded, replyTo(t, api, "/newtrip"))
	assert.Equal(t, msgEnterTripName, nextMessage(t, api))
	assertState(t, repos, models.StateNewTripName)

	list, err := ops.ListTripsByUser(ctx, repos, &models.User{ID: testUserID})
	require.NoError(t, err)
	require.Len(t, list, 1, "only the new draft is stored")
	assert.Empty(t, list[0].Name)

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}

func TestService_CancelCommand(t *testing.T) {
	tests := []struct {
		name      string
		steps     []string
		want      string
		wantState models.State
	}{
		{
			name:      "during the trip dialog",
			steps:     []string{"/newtrip", "Morning ride"},
			want:      msgDraftDiscarded,
			wantState: models.StateStart,
		},
		{
			name:      "at the confirmation",
			steps:     []string{"/newtrip", "Morning ride", "today", "Easy pace"},
			want:      msgDraftDiscarded,
			wantState: models.StateStart,
		},
		{
			name:      "outside of a dialog",
			want:      msgNothingToCancel,
			wantState: models.StateStart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			repos := newTestRepos()
			svc, api := newTestService(t, repos)

			svc.Start(ctx)

			for _, step := range tt.steps {
				replyTo(t, api, step)
			}

			assert.Equal(t, tt.want, replyTo(t, api, "/cancel"))
			assertState(t, repos, tt.wantState)

			assert.Equal(t, "You have no trips yet. Use /newtrip command to create a new trip.", replyTo(t, api, "/mytrips"))

			svc.Shutdown(ctx)
			assert.Equal(t, msgFarewell, nextMessage(t, api))
		})
	}
}

func TestService_ConfirmTrip(t *testing.T) {
	tests := []struct {
		name       string
		answers    []string
		wantPrefix string
		wantState  models.State
	}{
		{
			name:       "yes in another case",
			answers:    []string{"Yes"},
			wantPrefix: "Trip is published!",
			wantState:  models.StateStart,
		},
		{
			name:       "no with spaces",
			answers:    []string{" NO "},
			wantPrefix: "Your trip is canceled.",
			wantState:  models.StateNewTrip,
		},
		{
			name:       "unclear answer",
			answers:    []string{"maybe"},
			wantPrefix: msgConfirmAgain,
			wantState:  models.StateNewTripConfirm,
		},
		{
			name:       "yes after an unclear answer",
			answers:    []string{"maybe", "yes"},
			wantPrefix: "Trip is published!",
			wantState:  models.StateStart,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			repos := newTestRepos()
			svc, api := newTestService(t, repos)

			svc.Start(ctx)

			toConfirmation(t, api, repos)

			var reply string

			for _, answer := range tt.answers {
				reply = replyTo(t, api, answer)
			}

			assert.True(t, strings.HasPrefix(reply, tt.wantPrefix), reply)
			assertState(t, repos, tt.wantState)

			if tt.wantState == models.StateStart {
				assert.Equal(t, "pinChatMessage", api.NextRequest(t).Method)
			}

			svc.Shutdown(ctx)
			assert.Equal(t, msgFarewell, nextMessage(t, api))
		})
	}
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

func TestService_HelpListsEnabledCommands(t *testing.T) {
	ctx := context.Background()

	api := telegramtest.NewServer(t)
	// Telegram keeps its old menu; /help must still list the commands declared in the code.
	api.Fail("setMyCommands")

	// telegram.NewBot calls getMe, getMyDescription, getMyShortDescription, getMyCommands, and setMyCommands.
	svc, _ := newTestServiceWithAPI(t, newTestRepos(), api, 5, telegram.WithCommands(telegram.Commands{
		telegram.NewCommand(CmdStart, "start using the bot", true),
		telegram.NewCommand(CmdHelp, "show help", true),
		telegram.NewCommand(CmdTrips, "show all trips", false),
	}))

	svc.Start(ctx)

	help := replyTo(t, api, "/help")
	assert.Contains(t, help, "\t/start - start using the bot\n\t/help - show help\n")
	assert.NotContains(t, help, "/trips")

	svc.Shutdown(ctx)
	assert.Equal(t, msgFarewell, nextMessage(t, api))
}

func TestService_InterruptTrip(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "start",
			cmd:  "/" + CmdStart,
			want: "Hello, John!",
		},
		{
			name: "help",
			cmd:  "/" + CmdHelp,
			want: "Help!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			repos := newTestRepos()
			svc, api := newTestService(t, repos)

			svc.Start(ctx)

			replyTo(t, api, "/newtrip")
			replyTo(t, api, "Morning ride")
			assertState(t, repos, models.StateNewTripDate)

			assert.Equal(t, msgDraftDiscarded, replyTo(t, api, tt.cmd))
			assert.Contains(t, nextMessage(t, api), tt.want)
			assertState(t, repos, models.StateStart)

			assert.Equal(t, "You have no trips yet. Use /newtrip command to create a new trip.", replyTo(t, api, "/mytrips"))

			svc.Shutdown(ctx)
			assert.Equal(t, msgFarewell, nextMessage(t, api))
		})
	}
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
