package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Env           string `yaml:"env"`
		HTTPAddr      string `yaml:"http_addr"`
		PublicBaseURL string `yaml:"public_base_url"`
	} `yaml:"server"`
	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
	Wechat struct {
		AppID       string `yaml:"app_id"`
		AppSecret   string `yaml:"app_secret"`
		UseFakeAuth bool   `yaml:"use_fake_auth"`
	} `yaml:"wechat"`
	Auth struct {
		SigningKey    string `yaml:"signing_key"`
		TokenTTLHours int    `yaml:"token_ttl_hours"`
	} `yaml:"auth"`
	Realtime struct {
		WriteTimeoutSeconds int `yaml:"write_timeout_seconds"`
		PongTimeoutSeconds  int `yaml:"pong_timeout_seconds"`
		PingIntervalSeconds int `yaml:"ping_interval_seconds"`
		ClientSendQueueSize int `yaml:"client_send_queue_size"`
	} `yaml:"realtime"`
	Settlement struct {
		AutoCheckIntervalSeconds int `yaml:"auto_check_interval_seconds"`
		InactivityHours          int `yaml:"inactivity_hours"`
	} `yaml:"settlement"`
}

func Default() Config {
	var cfg Config
	cfg.Server.Env = "development"
	cfg.Server.HTTPAddr = ":8080"
	cfg.Server.PublicBaseURL = "http://localhost:8080"
	cfg.Database.Path = "./data/oneround.db"
	cfg.Wechat.UseFakeAuth = true
	cfg.Auth.SigningKey = "dev-only-change-me"
	cfg.Auth.TokenTTLHours = 720
	cfg.Realtime.WriteTimeoutSeconds = 10
	cfg.Realtime.PongTimeoutSeconds = 60
	cfg.Realtime.PingIntervalSeconds = 30
	cfg.Realtime.ClientSendQueueSize = 32
	cfg.Settlement.AutoCheckIntervalSeconds = 300
	cfg.Settlement.InactivityHours = 24
	return cfg
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				return cfg, err
			}
		}
	}
	applyEnv(&cfg)
	return cfg, nil
}

func (c Config) TokenTTL() time.Duration {
	return time.Duration(c.Auth.TokenTTLHours) * time.Hour
}

func (c Config) AutoCheckInterval() time.Duration {
	secs := c.Settlement.AutoCheckIntervalSeconds
	if secs <= 0 {
		secs = 300
	}
	return time.Duration(secs) * time.Second
}

func (c Config) InactivityThreshold() time.Duration {
	hours := c.Settlement.InactivityHours
	if hours <= 0 {
		hours = 24
	}
	return time.Duration(hours) * time.Hour
}

func applyEnv(cfg *Config) {
	setString := func(key string, dest *string) {
		if v := os.Getenv(key); v != "" {
			*dest = v
		}
	}
	setBool := func(key string, dest *bool) {
		if v := os.Getenv(key); v != "" {
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				*dest = parsed
			}
		}
	}
	setInt := func(key string, dest *int) {
		if v := os.Getenv(key); v != "" {
			parsed, err := strconv.Atoi(v)
			if err == nil {
				*dest = parsed
			}
		}
	}
	setString("ONEROUND_HTTP_ADDR", &cfg.Server.HTTPAddr)
	setString("ONEROUND_DATABASE_PATH", &cfg.Database.Path)
	setString("ONEROUND_WECHAT_APP_ID", &cfg.Wechat.AppID)
	setString("ONEROUND_WECHAT_APP_SECRET", &cfg.Wechat.AppSecret)
	setBool("ONEROUND_WECHAT_USE_FAKE_AUTH", &cfg.Wechat.UseFakeAuth)
	setString("ONEROUND_AUTH_SIGNING_KEY", &cfg.Auth.SigningKey)
	setInt("ONEROUND_AUTH_TOKEN_TTL_HOURS", &cfg.Auth.TokenTTLHours)
	setInt("ONEROUND_SETTLEMENT_AUTO_CHECK_INTERVAL_SECONDS", &cfg.Settlement.AutoCheckIntervalSeconds)
	setInt("ONEROUND_SETTLEMENT_INACTIVITY_HOURS", &cfg.Settlement.InactivityHours)
}
