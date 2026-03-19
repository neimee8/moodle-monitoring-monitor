package requests

type Response struct {
	Body       []byte
	StatusCode int
	FinalUrl   string
	Retries    int
	Err        error
}
