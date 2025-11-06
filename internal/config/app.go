package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type HTTPServer struct {
	Port string `mapstructure:"port"`
}

type DbServer struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
	User string `mapstructure:"user"`
	Pass string `mapstructure:"pass"`
	Name string `mapstructure:"name"`
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
}

func Init() *AppConfig {
	var cfg AppConfig

	if err := godotenv.Load(); err != nil {
		log.Fatalf("❌ Failed to load .env")
	}

	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("❌ Failed to parse in config: %v", err)
	}

	// db server
	_ = viper.BindEnv("db_server.host", "DB_HOST")
	_ = viper.BindEnv("db_server.port", "DB_PORT")
	_ = viper.BindEnv("db_server.user", "DB_USER")
	_ = viper.BindEnv("db_server.pass", "DB_PASS")
	_ = viper.BindEnv("db_server.name", "DB_NAME")

	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("❌ Failed to unmarshal config: %v", err)
	}

	return &cfg
}
