package settings

import (
	"encoding/json"
	"monitor/internal/config"
	"monitor/internal/requests"
	"net/http"
	"strconv"
)

var req *requests.Request

// Settings describes the runtime settings consumed by the monitor.
type Settings struct {
	AllowedTelegramChatIds             []string          `json:"allowed_telegram_chat_ids"`
	AdminTelegramChatIds               []string          `json:"admin_telegram_chat_ids"`
	ActiveTelegramChatIds              []string          `json:"active_telegram_chat_ids"`
	Courses                            map[string]string `json:"courses"`
	MoodleSessions                     map[string]string `json:"moodle_sessions"`
	MonitorRequestCycleCooldownSeconds uint              `json:"monitor_request_cycle_cooldown_seconds"`
	TelegramApiToken                   string            `json:"telegram_api_token"`
	WebsettingsLogLineCount            uint              `json:"websettings_log_line_count"`
}

// response mirrors the settings API response envelope.
type response struct {
	Msg  string   `json:"msg"`
	Data Settings `json:"data"`
}

// initialize prepares the shared request used to fetch settings.
func initialize(cfg *config.Config) {
	if req == nil {
		req = requests.NewRequest(cfg)
		req.Url = cfg.GetSettingsApiEndpoint
	}
}

// Load fetches the current monitor settings from the settings API.
func Load(cfg *config.Config) *Settings {
	initialize(cfg)

	resp := req.Do()

	if resp.Err != nil {
		panic("load settings error: request error: " + resp.Err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		panic("load settings error: bad response code: " + strconv.Itoa(resp.StatusCode))
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
