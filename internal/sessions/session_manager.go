package sessions

import (
	"fmt"
	"monitor/internal/settings"
	"monitor/internal/types"
	"sync"
)

type SessionManager struct {
	mu               sync.Mutex
	sessions         Sessions
	timedOutSessions types.Set[Session]
	lastUsedIdIndex  int
}

func NewSessionManager(stg *settings.Settings) *SessionManager {
	sessions := make(Sessions, 0, len(stg.MoodleSessions))

	for id, value := range stg.MoodleSessions {
		sessions = append(sessions, Session{
			Id:    id,
			Value: value,
		})
	}

	return &SessionManager{
		sessions:         sessions,
		timedOutSessions: types.NewSet[Session](),
	}
}

func (s *SessionManager) GetSession() (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var session Session

	if len(s.sessions) == 0 || s.timedOutSessions.Size() == len(s.sessions) {
		return session, NoValidSessionsError
	}

	idx := s.lastUsedIdIndex

	for true {
		if s.lastUsedIdIndex == len(s.sessions)-1 {
			s.lastUsedIdIndex = 0
		} else {
			s.lastUsedIdIndex++
		}

		if !s.timedOutSessions.Exists(s.sessions[s.lastUsedIdIndex]) {
			idx = s.lastUsedIdIndex
			break
		}
	}

	session = s.sessions[idx]
	return session, nil
}

func (s *SessionManager) GetTimedOutSessions() Sessions {
	return Sessions(s.timedOutSessions.ToSlice())
}

func (s *SessionManager) TimedOut(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false

	for _, s := range s.sessions {
		if s == session {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("given session (id: %s) does not match any registered session", session.Id)
	}

	s.timedOutSessions.Add(session)
	return nil
}
