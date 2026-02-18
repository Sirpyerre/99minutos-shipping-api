package config

import (
	"context"
	"log/slog"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port      string `env:"PORT,      default=8080"`
	Env       string `env:"ENV,       default=development"`
	JWTSecret string `env:"JWT_SECRET"`
	LogLevel  string `env:"LOG_LEVEL, default=info"`

	Mongo MongoConfig
	Redis RedisConfig
}

type MongoConfig struct {
	URI      string `env:"MONGO_URI, default=mongodb://localhost:27017"`
	Database string `env:"MONGO_DB,  default=shipping_system"`
}

type RedisConfig struct {
	Addr string `env:"REDIS_ADDR, default=localhost:6379"`
	DB   int    `env:"REDIS_DB,   default=0"`
}



// Load reads configuration from environment variables using go-envconfig.
func Load(logger *slog.Logger) *Config {
	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		logger.Error("Failed to load configuration", "error", err)
		panic(err)
	}
	return &cfg
}
