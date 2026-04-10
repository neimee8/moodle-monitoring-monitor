package sessions

import (
	"fmt"
	"monitor/internal/utils"
)

// Session stores a Moodle session identifier and cookie value.
type Session struct {
	Id    string
	Value string
}

// Sessions is a list of Moodle sessions.
type Sessions []Session

// Repr returns a plain-text representation of the session.
func (s Session) Repr() string {
	return fmt.Sprintf(
		"%s: %s\n",
		s.Id,
		s.Value,
	)
}

// Repr returns a plain-text representation of all sessions.
func (s Sessions) Repr() string {
	repr := ""

	for _, session := range s {
		repr += session.Repr() + "\n"
	}

	return repr
}

// Diff returns sessions added in b and removed from a.
func (a Sessions) Diff(b Sessions) (added, removed Sessions) {
	return utils.SliceDiffComparable(a, b)
}
