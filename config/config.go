package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"time"
)

type Config struct {
	HTTPAddr          string        `env:"DNS_MANAGER_HTTP_ADDR" env-default:":8080"`
	ResolvConfPath    string        `env:"DNS_MANAGER_RESOLV_CONF_PATH" env-default:"/etc/resolv.conf"`
	ReadHeaderTimeout time.Duration `env:"DNS_MANAGER_READ_HEADER_TIMEOUT" env-default:"5s"`
	ShutdownTimeout   time.Duration `env:"DNS_MANAGER_SHUTDOWN_TIMEOUT" env-default:"10s"`
	LogLevel          string        `env:"DNS_MANAGER_LOG_LEVEL" env-default:"info"`
	LogFilePath       string        `env:"DNS_MANAGER_LOG_FILE_PATH" env-default:"logs/dns-server.log"`
}

func LoadConfig() *Config {
	var cfg Config

	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			log.Fatalf("Cannot read config: %s", err)
		}
	}

	return &cfg
}
