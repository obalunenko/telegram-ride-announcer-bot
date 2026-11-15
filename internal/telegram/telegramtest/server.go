// Package telegramtest provides a fake Telegram Bot API server for tests.
package telegramtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/mymmrac/telego"
)

const (
	// Token is a bot token that passes the telego token validation.
	Token = "123456789:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	// BotID is the ID of the bot returned by getMe.
	BotID int64 = 123456789
	// BotUsername is the username of the bot returned by getMe.
	BotUsername = "test_bot"

	// pollTimeout is how long getUpdates waits for an update, like the real long polling.
	pollTimeout = 100 * time.Millisecond
	// waitTimeout is how long NextRequest waits for a Bot API call.
	waitTimeout = 5 * time.Second
	// queueSize is how many calls and updates the Server holds before it blocks.
	queueSize = 100
)

// Request is a Bot API method call received by the Server.
type Request struct {
	Method string
	Params map[string]any
}

// Server is a fake Telegram Bot API server.
// It records every call except getUpdates and serves updates queued with SendUpdate to long polling.
type Server struct {
	srv *httptest.Server

	requests chan Request
	updates  chan tgbotapi.Update

	mu        sync.Mutex
	results   map[string]any
	failures  map[string]bool
	updateID  int
	messageID int
}

type response struct {
	Ok          bool   `json:"ok"`
	Result      any    `json:"result,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
}

// NewServer starts a Server that is closed when the test finishes.
func NewServer(t testing.TB) *Server {
	t.Helper()

	s := &Server{
		requests: make(chan Request, queueSize),
		updates:  make(chan tgbotapi.Update, queueSize),
		results: map[string]any{
			"getMe": tgbotapi.User{
				ID:        BotID,
				IsBot:     true,
				FirstName: "Test Bot",
				Username:  BotUsername,
			},
			"getMyDescription":      tgbotapi.BotDescription{},
			"getMyShortDescription": tgbotapi.BotShortDescription{},
			"getMyCommands":         []tgbotapi.BotCommand{},
		},
		failures: make(map[string]bool),
	}

	s.srv = httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	t.Cleanup(s.srv.Close)

	return s
}

// ClientOptions returns telego options that point a bot client to the Server.
func (s *Server) ClientOptions() []tgbotapi.BotOption {
	return []tgbotapi.BotOption{
		tgbotapi.WithAPIServer(s.srv.URL),
		tgbotapi.WithHTTPClient(s.srv.Client()),
		tgbotapi.WithDiscardLogger(),
	}
}

// SetResult sets the result the Server returns for the method.
func (s *Server) SetResult(method string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.results[method] = result
}

// Fail makes the Server return an error for every call of the method.
func (s *Server) Fail(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failures[method] = true
}

// SendUpdate queues the update for long polling and assigns it the next update ID.
func (s *Server) SendUpdate(update tgbotapi.Update) {
	s.mu.Lock()
	s.updateID++
	update.UpdateID = s.updateID
	s.mu.Unlock()

	s.updates <- update
}

// NextRequest returns the next recorded call. It fails the test if no call arrives in time.
func (s *Server) NextRequest(t testing.TB) Request {
	t.Helper()

	select {
	case req := <-s.requests:
		return req
	case <-time.After(waitTimeout):
		t.Fatal("telegramtest: no Bot API call received")

		return Request{}
	}
}

// AssertNoRequest fails the test if there is a recorded call that was not read with NextRequest.
func (s *Server) AssertNoRequest(t testing.TB) {
	t.Helper()

	select {
	case req := <-s.requests:
		t.Errorf("telegramtest: unexpected Bot API call %q", req.Method)
	default:
	}
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	method := path.Base(r.URL.Path)

	var params map[string]any

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeResponse(w, response{Ok: false, ErrorCode: http.StatusBadRequest, Description: err.Error()})

		return
	}

	if method == "getUpdates" {
		s.serveUpdates(w, r)

		return
	}

	s.requests <- Request{Method: method, Params: params}

	result, ok := s.result(method, params)
	if !ok {
		writeResponse(w, response{Ok: false, ErrorCode: http.StatusBadRequest, Description: "Bad Request: " + method})

		return
	}

	writeResponse(w, response{Ok: true, Result: result})
}

func (s *Server) serveUpdates(w http.ResponseWriter, r *http.Request) {
	var updates []tgbotapi.Update

	select {
	case update := <-s.updates:
		updates = append(updates, update)
	case <-time.After(pollTimeout):
	case <-r.Context().Done():
	}

	writeResponse(w, response{Ok: true, Result: updates})
}

func (s *Server) result(method string, params map[string]any) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failures[method] {
		return nil, false
	}

	if result, ok := s.results[method]; ok {
		return result, true
	}

	if method == "sendMessage" {
		s.messageID++

		return map[string]any{
			"message_id": s.messageID,
			"date":       0,
			"chat":       map[string]any{"id": params["chat_id"], "type": "private"},
			"text":       params["text"],
		}, true
	}

	return true, true
}

func writeResponse(w http.ResponseWriter, resp response) {
	w.Header().Set("Content-Type", "application/json")

	if !resp.Ok {
		w.WriteHeader(resp.ErrorCode)
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
