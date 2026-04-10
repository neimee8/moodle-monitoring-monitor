package sessions

import "errors"

// NoValidSessionsError indicates that no usable sessions remain.
var NoValidSessionsError = errors.New("no valid sessions")
