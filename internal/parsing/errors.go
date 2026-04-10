package parsing

import "errors"

// RequestInterruptedError indicates that parsing stopped because the request was interrupted.
var RequestInterruptedError = errors.New("parsing error: request has been interrupted")
