package requests

import (
	"io"
	"math"
	"math/rand"
	"monitor/internal/config"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Request struct {
	cfg    *config.Config
	client *http.Client

	Url            string
	Method         string
	Headers        map[string][]string
	Queries        map[string][]string
	Cookies        map[string]string
	Body           string
	TimeoutSeconds uint
	Retries        uint
	Semaphore      *Semaphore

	InterruptRequestCallback func() bool
}

func NewRequest(cfg *config.Config) *Request {
	return &Request{
		cfg:    cfg,
		client: &http.Client{},

		Method:         "GET",
		Headers:        make(map[string][]string),
		Queries:        make(map[string][]string),
		Cookies:        make(map[string]string),
		TimeoutSeconds: cfg.DefaultTimeoutSeconds,
	}
}

func (r *Request) Do() *Response {
	if r.InterruptRequestCallback != nil && r.InterruptRequestCallback() {
		return &Response{
			Err: RequestInterruptedError,
		}
	}

	if r.Semaphore != nil {
		r.Semaphore.Acquire()
		defer r.Semaphore.Release()
	}

	u, err := url.Parse(r.Url)

	if err != nil {
		return &Response{
			Err: err,
		}
	}

	q := u.Query()

	for param, values := range r.Queries {
		for _, value := range values {
			q.Add(param, value)
		}
	}

	u.RawQuery = q.Encode()

	var bodyReader io.Reader

	if r.Body != "" {
		bodyReader = strings.NewReader(r.Body)
	}

	req, err := http.NewRequest(
		strings.ToUpper(r.Method),
		u.String(),
		bodyReader,
	)

	if err != nil {
		return &Response{
			Err: err,
		}
	}

	for header, values := range r.Headers {
		for _, value := range values {
			req.Header.Add(header, value)
		}
	}

	for cookie, value := range r.Cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie,
			Value: value,
		})
	}

	if r.client == nil {
		r.client = &http.Client{}
	}

	r.client.Timeout = time.Duration(r.TimeoutSeconds) * time.Second

	var resp *http.Response
	var retryIdx int

	for retryIdx = 0; retryIdx <= int(r.Retries); retryIdx++ {
		if r.InterruptRequestCallback != nil && r.InterruptRequestCallback() {
			return &Response{
				Err: RequestInterruptedError,
			}
		}

		delay := r.calculateRetryDelay(retryIdx)

		for i := 0; i < 10 && delay > 0; i++ {
			if r.InterruptRequestCallback != nil && r.InterruptRequestCallback() {
				return &Response{
					Err: RequestInterruptedError,
				}
			}

			time.Sleep(time.Duration(max(delay/10, 1)) * time.Millisecond)
		}

		resp, err = r.client.Do(req)
		shouldRetry := true

		if err == nil {
			shouldRetry = false

			for _, code := range r.cfg.RetryStatusCodes {
				if code == resp.StatusCode {
					shouldRetry = true
					break
				}
			}
		}

		if !shouldRetry {
			break
		}
	}

	if err != nil {
		return &Response{
			Err: err,
		}
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return &Response{
			Err: err,
		}
	}

	return &Response{
		Body:       body,
		StatusCode: resp.StatusCode,
		FinalUrl:   resp.Request.URL.String(),
		Retries:    retryIdx,
	}
}

func (r *Request) calculateRetryDelay(retryIdx int) int {
	if retryIdx == 0 {
		return 0
	}

	delay := min(
		r.cfg.BaseRetryDelayMilliseconds*math.Pow(2, float64(retryIdx-1)),
		r.cfg.MaxRetryDelayMilliseconds,
	)

	jitter := r.cfg.MinRetryJitterMultiplier +
		rand.Float64()*(r.cfg.MaxRetryJitterMultiplier-r.cfg.MinRetryJitterMultiplier)

	return int(delay * jitter)
}

func (r *Request) DeepCopy() *Request {
	headers := make(map[string][]string)
	queries := make(map[string][]string)
	cookies := make(map[string]string)

	for header, values := range r.Headers {
		headers[header] = make([]string, 0, len(values))

		for _, value := range values {
			headers[header] = append(headers[header], value)
		}
	}

	for query, values := range r.Queries {
		queries[query] = make([]string, 0, len(values))

		for _, value := range values {
			queries[query] = append(queries[query], value)
		}
	}

	for cookie, value := range r.Cookies {
		cookies[cookie] = value
	}

	return &Request{
		cfg:    r.cfg,
		client: r.client,

		Url:            r.Url,
		Method:         r.Method,
		Headers:        headers,
		Queries:        queries,
		Cookies:        cookies,
		Body:           r.Body,
		TimeoutSeconds: r.TimeoutSeconds,
		Retries:        r.Retries,
		Semaphore:      r.Semaphore,

		InterruptRequestCallback: r.InterruptRequestCallback,
	}
}
