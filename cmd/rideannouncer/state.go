package main

import "sync"

type state uint

const (
	stateUnknown state = iota
	stateStart
	stateNewTrip
	stateNewTripName
	stateNewTripDate
	stateNewTripTime
	stateNewTripDescription
	stateNewTripConfirm
	stateNewTripConfirmYes
	stateNewTripConfirmNo
)

type session struct {
	user   *user
	chatID int64
	state  state
}

type user struct {
	id        int64
	username  string
	firstname string
	lastname  string
}

func newUser(id int64, username, firstname, lastname string) *user {
	return &user{
		id:        id,
		username:  username,
		firstname: firstname,
		lastname:  lastname,
	}
}

func newSession(u *user, chatID int64) *session {
	return &session{
		user:   u,
		chatID: chatID,
		state:  stateUnknown,
	}
}

type (
	chatID = int64
	userID = int64
)

var (
	sessions = make(map[userID]*session)
	chatIDs  map[chatID]struct{}
	mu       sync.RWMutex
)

func getSession(uid userID) *session {
	mu.Lock()
	defer mu.Unlock()

	s, exist := sessions[uid]
	if !exist {
		return nil
	}

	return s
}

func setSession(sess *session, uid userID) {
	mu.Lock()
	defer mu.Unlock()

	sessions[uid] = sess
}

func deleteSession(userID int64) {
	mu.Lock()
	defer mu.Unlock()

	delete(sessions, userID)
}

func (s *session) reset() {
	s.state = stateUnknown
}

func (s *session) setState(st state) {
	s.state = st
}

func (s *session) getState() state {
	return s.state
}

func (s *session) isState(st state) bool {
	return s.state == st
}

func (s *session) isStateAny(sts ...state) bool {
	for _, st := range sts {
		if s.isState(st) {
			return true
		}
	}

	return false
}

func (s *session) isStateNot(st state) bool {
	return !s.isState(st)
}

func (s *session) isStateNotAny(sts ...state) bool {
	for _, st := range sts {
		if s.isState(st) {
			return false
		}
	}

	return true
}
