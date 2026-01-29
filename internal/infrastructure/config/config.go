package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type DatabaseConfig struct {
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	Host     string `env:"POSTGRES_HOST" envDefault:"postgres"`
	Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
	Name     string `env:"POSTGRES_DB" envDefault:"orders_db"`
}

type KafkaConfig struct {
	Broker  string `env:"KAFKA_BROKER" envDefault:"localhost:9092"`
	Topic   string `env:"KAFKA_TOPIC" envDefault:"orders"`
	GroupID string `env:"KAFKA_GROUP_ID" envDefault:"orders_group"`
}

type RedisConfig struct {
	Addr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
}

type HTTPConfig struct {
	Port            string        `env:"HTTP_PORT" envDefault:"8080"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

type ProducerConfig struct {
	Kafka KafkaConfig
}

type ConsumerConfig struct {
	Kafka    KafkaConfig
	Database DatabaseConfig
	Redis    RedisConfig
	HTTP     HTTPConfig
}

func LoadProducerConfig() (*ProducerConfig, error) {
	cfg := &ProducerConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadConsumerConfig() (*ConsumerConfig, error) {
	cfg := &ConsumerConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	if cfg.Database.User == "" || cfg.Database.Password == "" {
		return nil, fmt.Errorf("database credentials required for consumer")
	}

	return cfg, nil
}
