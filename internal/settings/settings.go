package settings

import (
	"encoding/json"
	"monitor/internal/config"
	"monitor/internal/requests"
	"net/http"
)

var req *requests.Request

type Settings struct {
	AllowedTelegramChatIds             []string          `json:"allowed_telegram_chat_ids"`
	AdminTelegramChatIds               []string          `json:"admin_telegram_chat_ids"`
	ActiveTelegramChatIds              []string          `json:"active_telegram_chat_ids"`
	Courses                            map[string]string `json:"courses"`
	MoodleSession                      string            `json:"moodle_session"`
	MonitorRequestCycleCooldownSeconds uint              `json:"monitor_request_cycle_cooldown_seconds"`
	TelegramApiToken                   string            `json:"telegram_api_token"`
	WebsettingsLogLineCount            uint              `json:"websettings_log_line_count"`
}

type response struct {
	Msg  string   `json:"msg"`
	Data Settings `json:"data"`
}

func initialize(cfg *config.Config) {
	if req == nil {
		req = requests.NewRequest(cfg)
		req.Url = cfg.GetSettingsApiEndpoint
	}
}

func Load(cfg *config.Config) *Settings {
	initialize(cfg)

	resp := req.Do()

	if resp.Err != nil {
		panic("load settings error: request error: " + resp.Err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		panic("load settings error: bad response code: " + string(resp.StatusCode))
	}

	var r response
	err := json.Unmarshal(resp.Body, &r)

	if err != nil {
		panic("load settings error: json unmarshal error: " + err.Error())
	}

	if r.Msg != cfg.ApiResponseMsgOk {
		panic("load settings error: request is not successful. response message: " + r.Msg)
	}

	return &r.Data
}
