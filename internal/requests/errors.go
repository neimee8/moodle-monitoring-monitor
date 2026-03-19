package requests

import "errors"

var RequestInterruptedError = errors.New("request interrupted by callback")
