package sessions

import (
	"fmt"
	"monitor/internal/utils"
)

type Session struct {
	Id    string
	Value string
}

type Sessions []Session

func (s Session) Repr() string {
	return fmt.Sprintf(
		"%s: %s\n",
		s.Id,
		s.Value,
	)
}

func (s Sessions) Repr() string {
	repr := ""

	for _, session := range s {
		repr += session.Repr() + "\n"
	}

	return repr
}

func (a Sessions) Diff(b Sessions) (added, removed Sessions) {
	return utils.SliceDiffComparable(a, b)
}
