package messages

import (
	"encoding/json"
	"fmt"
	"monitor/internal/config"
	"monitor/internal/requests"
	"net/http"
	"strconv"
	"strings"
)

// Sender sends notifications through the messaging API.
type Sender struct {
	cfg *config.Config
	req *requests.Request
}

// NewSender returns a sender configured for the notification endpoint.
func NewSender(cfg *config.Config, interruptRequestCallback func() bool) *Sender {
	req := requests.NewRequest(cfg)
	req.Url = cfg.SendMsgApiEndpoint
	req.Method = "POST"
	req.Retries = cfg.SendMsgRetries
	req.InterruptRequestCallback = interruptRequestCallback

	return &Sender{
		cfg: cfg,
		req: req,
	}
}

// Do sends a message and optionally targets admin recipients only.
func (s Sender) Do(text string, admin bool) error {
	text = strings.TrimSpace(text)
	adminStr := "0"

	if admin {
		adminStr = "1"
	}

	body := map[string]string{
		"target":     "",
		"text":       text,
		"parse_mode": "html",
		"admin":      adminStr,
		"force":      "0",
	}

	bodyJson, err := json.Marshal(body)

	if err != nil {
		panic("send message error: request body marshal error: " + err.Error())
	}

	s.req.Body = string(bodyJson)
	resp := s.req.Do()

	if resp.Err != nil {
		panic("send message error: request error: " + resp.Err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf(
			"send message error: bad response code: %s. response: %s",
			strconv.Itoa(resp.StatusCode),
			resp.Body,
		))
	}

	var respBody map[string]string
	err = json.Unmarshal(resp.Body, &respBody)

	if err != nil {
		panic("send message error: response body unmarshal error: " + err.Error())
	}

	respMsg := respBody["msg"]

	if respMsg != s.cfg.ApiResponseMsgOk {
		panic("send message error: request is not successful. response message: " + respMsg)
	}

	return nil
}
