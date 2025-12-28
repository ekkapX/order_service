package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	HTTPServerPort  string        `env:"HTTP_PORT" envDefault:"8080"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`

	Database struct {
		User     string `env:"POSTGRES_USER,required"`
		Password string `env:"POSTGRES_PASSWORD,required"`
		Host     string `env:"POSTGRES_HOST" envDefault:"postgres"`
		Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
		Name     string `env:"POSTGRES_DB" envDefault:"orders_db"`
	}

	Kafka struct {
		Broker  string `env:"KAFKA_BROKER" envDefault:"localhost:9092"`
		Topic   string `env:"KAFKA_TOPIC" envDefault:"orders"`
		GroupID string `env:"KAFKA_GROUP_ID" envDefault:"orders_group"`
	}

	Redis struct {
		Addr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	}
}

func LoadConfig() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
