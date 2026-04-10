package requests

import "errors"

// RequestInterruptedError indicates that a request was canceled by the interrupt callback.
var RequestInterruptedError = errors.New("request interrupted by callback")
