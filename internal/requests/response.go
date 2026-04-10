package requests

// Response contains the result of an executed request.
type Response struct {
	Body       []byte
	StatusCode int
	FinalUrl   string
	Retries    int
	Err        error
}
