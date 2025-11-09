package config

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
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
		logrus.Fatalf("Error loading .env file: %s", err) // todo: handle
	}

	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalf("Error reading config file, %s", err) // todo: handle
	}

	// db server
	_ = viper.BindEnv("db_server.host", "DB_HOST")
	_ = viper.BindEnv("db_server.port", "DB_PORT")
	_ = viper.BindEnv("db_server.user", "DB_USER")
	_ = viper.BindEnv("db_server.pass", "DB_PASS")
	_ = viper.BindEnv("db_server.name", "DB_NAME")

	if err := viper.Unmarshal(&cfg); err != nil {
		logrus.Fatalf("Error unmarshalling config: %s", err) // todo: handle
	}

	return &cfg
}

func LoadSupportedCurrencies(ctx context.Context, pool *pgxpool.Pool) (map[string]struct{}, error) {
	rows, err := pool.Query(ctx, `select code from currencies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]struct{})
	for rows.Next() {
		var c string
		if err = rows.Scan(&c); err != nil {
			return nil, err
		}
		m[c] = struct{}{}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return m, nil
}
