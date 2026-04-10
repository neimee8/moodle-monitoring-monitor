package sessions

import (
	"fmt"
	"monitor/internal/settings"
	"monitor/internal/types"
	"sync"
)

// SessionManager rotates through Moodle sessions and tracks timed-out ones.
type SessionManager struct {
	mu               sync.Mutex
	sessions         Sessions
	timedOutSessions types.Set[Session]
	lastUsedIdIndex  int
}

// NewSessionManager returns a session manager built from the configured Moodle sessions.
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

// GetSession returns the next available session in round-robin order.
func (s *SessionManager) GetSession() (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var session Session

	if len(s.sessions) == 0 || s.timedOutSessions.Size() == len(s.sessions) {
		return session, NoValidSessionsError
	}

	idx := s.lastUsedIdIndex

	// Advance until a non-timed-out session is found.
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

// GetTimedOutSessions returns the sessions marked as timed out.
func (s *SessionManager) GetTimedOutSessions() Sessions {
	return Sessions(s.timedOutSessions.ToSlice())
}

// TimedOut marks a session as timed out after confirming that it belongs to the manager.
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
