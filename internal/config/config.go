package config

import (
	"fmt"
	"time"

	cleanenvport "github.com/wb-go/wbf/config/cleanenv-port"
	"github.com/wb-go/wbf/logger"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Postgres PostgresConfig `yaml:"postgres"`
	Logger   LoggerConfig   `yaml:"logger"`
	Gin      GinConfig      `yaml:"gin"`
	Retry    RetryConfig    `yaml:"retry"`
}

type ServerConfig struct {
	Addr            string        `yaml:"addr"          env:"SERVER_ADDR"`
	ReadTimeout     time.Duration `yaml:"read_timeout"  env:"SERVER_READ_TIMEOUT"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"  env:"SERVER_IDLE_TIMEOUT"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT"`
}

type LoggerConfig struct {
	Engine string `yaml:"engine" env:"LOG_ENGINE" env-default:"slog"`
	Level  string `yaml:"level"  env:"LOG_LEVEL"  env-default:"info"`
}

func (c LoggerConfig) LogLevel() logger.Level {
	switch c.Level {
	case "debug":
		return logger.DebugLevel
	case "warn":
		return logger.WarnLevel
	case "error":
		return logger.ErrorLevel
	default:
		return logger.InfoLevel
	}
}

func (c LoggerConfig) LogEngine() logger.Engine {
	return logger.Engine(c.Engine)
}

type GinConfig struct {
	Mode string `yaml:"mode" env:"GIN_MODE" env-default:"debug"`
}

type RetryConfig struct {
	Attempts int           `yaml:"attempts" env:"RETRY_ATTEMPTS"`
	Delay    time.Duration `yaml:"delay"    env:"RETRY_DELAY"`
	Backoff  float64       `yaml:"backoff"  env:"RETRY_BACKOFF"`
}

type PostgresConfig struct {
	Host            string        `yaml:"host"              env:"DB_HOST"`
	Port            int           `yaml:"port"              env:"DB_PORT"`
	User            string        `yaml:"user"              env:"DB_USER"`
	Password        string        `yaml:"password"          env:"DB_PASSWORD"`
	Database        string        `yaml:"database"          env:"DB_NAME"`
	SSLMode         string        `yaml:"sslmode"           env:"DB_SSLMODE"`
	MaxOpenConns    int           `yaml:"max_open_conns"    env:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `yaml:"max_idle_conns"    env:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME"`
}

func (p *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, p.SSLMode,
	)
}

func MustLoad() *Config {
	var cfg Config
	if err := cleanenvport.Load(&cfg); err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return &cfg
}
