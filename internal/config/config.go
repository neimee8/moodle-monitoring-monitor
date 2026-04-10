package config

import "os"

// Config stores static configuration values used across the monitor.
type Config struct {
	SendMsgApiEndpoint     string
	GetSettingsApiEndpoint string

	MoodleBaseUrl string
	MoodleHost    string

	DefaultTimeoutSeconds uint
	MoodleRequestRetries  uint
	SendMsgRetries        uint

	RetryStatusCodes           []int
	BaseRetryDelayMilliseconds float64
	MaxRetryDelayMilliseconds  float64
	MinRetryJitterMultiplier   float64
	MaxRetryJitterMultiplier   float64
	MaxConcurrentRequests      int

	SnapshotChannelBufferSize int

	LoggedOutMsgCooldownSeconds int

	ApiResponseMsgOk        string
	MoodleSessionCookieName string
	MoodleUserAgentHeader   string
	Sep                     string

	TimeFormat string
	FilePerm   os.FileMode

	StatePath    string
	StatePathTmp string
}

// Load returns the default application configuration.
func Load() *Config {
	return &Config{
		SendMsgApiEndpoint:     "http://localhost:8001/api/bot/send",
		GetSettingsApiEndpoint: "http://localhost:8000/api/settings",

		MoodleBaseUrl: "https://estudijas.rtu.lv",
		MoodleHost:    "estudijas.rtu.lv",

		DefaultTimeoutSeconds: 10,
		MoodleRequestRetries:  10,
		SendMsgRetries:        5,

		RetryStatusCodes: []int{
			500, 502, 503, 504, 429,
		},
		BaseRetryDelayMilliseconds: 100.0,
		MaxRetryDelayMilliseconds:  5000.0,
		MinRetryJitterMultiplier:   0.5,
		MaxRetryJitterMultiplier:   1.5,
		MaxConcurrentRequests:      50,

		SnapshotChannelBufferSize: 128,

		LoggedOutMsgCooldownSeconds: 150,

		ApiResponseMsgOk:        "ok",
		MoodleSessionCookieName: "MoodleSession",
		MoodleUserAgentHeader:   "Mozilla/5.0",
		Sep:                     "--------------------------------\n\n",

		TimeFormat: "02-01-2006 15:04:05.000",
		FilePerm:   0644,

		StatePath:    "/var/lib/moodle-monitoring/monitor-state.gob",
		StatePathTmp: "/var/lib/moodle-monitoring/monitor-state.gob.tmp",
	}
}
