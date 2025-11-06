package config

import "fmt"

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

func (config *DbServer) GetFormattedParams() string {
	return fmt.Sprintf(
		"user=%s host=%s port=%s dbname=%s",
		config.User, config.Host, config.Port, config.Name,
	)
}

type AppConfig struct {
	HTTPServer HTTPServer `mapstructure:"http_server"`
	DbServer   DbServer   `mapstructure:"db_server"`
}

func Init() *AppConfig {
	var cfg AppConfig
	return &cfg
}
