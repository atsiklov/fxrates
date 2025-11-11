package config

import (
	"fmt"
	"os"

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
	HTTPServer      HTTPServer      `mapstructure:"http_server"`
	DbServer        DbServer        `mapstructure:"db_server"`
	HTTPClient      HTTPClient      `mapstructure:"http_client"`
	ExchangeRateAPI ExchangeRateAPI `mapstructure:"exchange_rate_api"`
	Logging         Logging         `mapstructure:"logging"`
	Scheduler       Scheduler       `mapstructure:"scheduler"`
}

type HTTPClient struct {
	TimeoutSeconds int `mapstructure:"timeout_seconds"`
}

type Logging struct {
	Level string `mapstructure:"level"`
}

type ExchangeRateAPI struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
}

type Scheduler struct {
	JobDurationSec int `mapstructure:"job_duration_sec"`
}

func Init() (*AppConfig, error) {
	var cfg AppConfig

	if os.Getenv("PROFILE") != "" {
		// env vars will be passed from outside
	} else {
		// local: require .env
		if err := godotenv.Load(); err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// db server env vars
	_ = viper.BindEnv("db_server.host", "DB_HOST")
	_ = viper.BindEnv("db_server.port", "DB_PORT")
	_ = viper.BindEnv("db_server.user", "DB_USER")
	_ = viper.BindEnv("db_server.pass", "DB_PASS")
	_ = viper.BindEnv("db_server.name", "DB_NAME")
	_ = viper.BindEnv("db_server.max_conns", "DB_MAX_CONNS")

	// http client env vars
	_ = viper.BindEnv("http_client.timeout_seconds", "HTTP_CLIENT_TIMEOUT_SECONDS")
	// logging env var
	_ = viper.BindEnv("logging.level", "LOG_LEVEL")
	// exchange rate api env vars
	_ = viper.BindEnv("exchange_rate_api.base_url", "EXCHANGE_RATE_API_BASE_URL")
	_ = viper.BindEnv("exchange_rate_api.api_key", "EXCHANGE_RATE_API_KEY")

	// scheduler env vars
	_ = viper.BindEnv("scheduler.job_duration_sec", "JOB_DURATION_SEC")

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
