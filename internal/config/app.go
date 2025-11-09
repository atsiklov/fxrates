package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type HTTPServer struct {
	Port string `mapstructure:"port"`
}

type DbServer struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Pass     string `mapstructure:"pass"`
	Name     string `mapstructure:"name"`
	MaxConns int32  `mapstructure:"max_conns"`
}

func (config *DbServer) GetConnectionStr() string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable pool_max_conns=10",
		config.User, config.Pass, config.Host, config.Port, config.Name,
	)
}

type AppConfig struct {
	HTTPServer HTTPServer `mapstructure:"http_server"`
	DbServer   DbServer   `mapstructure:"db_server"`
	HTTPClient HTTPClient `mapstructure:"http_client"`
}

type HTTPClient struct {
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
}

func Init() (*AppConfig, error) {
	var cfg AppConfig

	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	viper.SetDefault("db_server.max_conns", 10)
	viper.SetDefault("http_client.timeout_seconds", 10)

	// db server env vars
	_ = viper.BindEnv("db_server.host", "DB_HOST")
	_ = viper.BindEnv("db_server.port", "DB_PORT")
	_ = viper.BindEnv("db_server.user", "DB_USER")
	_ = viper.BindEnv("db_server.pass", "DB_PASS")
	_ = viper.BindEnv("db_server.name", "DB_NAME")
	_ = viper.BindEnv("db_server.max_conns", "DB_MAX_CONNS")

	// http client env vars
	_ = viper.BindEnv("http_client.timeout_seconds", "HTTP_CLIENT_TIMEOUT_SECONDS")

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
