package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	S3       S3Config
	JWT      JWTConfig
	Email    EmailConfig
}

type AppConfig struct {
	Env  string
	Port string
	Host string
}

type DatabaseConfig struct {
	User     string
	Password string
	DBName   string
	Host     string
	Port     string
}

type RedisConfig struct {
	Host string
	Port string
}

type S3Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	UseSSL     bool
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpire  time.Duration
	RefreshExpire time.Duration
}

type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
}

func Load() (*Config, error) {
	viper.SetConfigFile("../.env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	accessExpire, _ := time.ParseDuration(viper.GetString("JWT_ACCESS_EXPIRE"))
	refreshExpire, _ := time.ParseDuration(viper.GetString("JWT_REFRESH_EXPIRE"))

	cfg := &Config{
		App: AppConfig{
			Env:  viper.GetString("APP_ENV"),
			Port: viper.GetString("APP_PORT"),
			Host: viper.GetString("APP_HOST"),
		},
		Database: DatabaseConfig{
			User:     viper.GetString("POSTGRES_USER"),
			Password: viper.GetString("POSTGRES_PASSWORD"),
			DBName:   viper.GetString("POSTGRES_DB"),
			Host:     "localhost",
			Port:     viper.GetString("POSTGRES_PORT"),
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: viper.GetString("REDIS_PORT"),
		},
		S3: S3Config{
			Endpoint:   fmt.Sprintf("localhost:%s", viper.GetString("MINIO_API_PORT")),
			AccessKey:  viper.GetString("MINIO_ROOT_USER"),
			SecretKey:  viper.GetString("MINIO_ROOT_PASSWORD"),
			BucketName: viper.GetString("S3_BUCKET_NAME"),
			Region:     viper.GetString("S3_REGION"),
			UseSSL:     false,
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpire:  accessExpire,
			RefreshExpire: refreshExpire,
		},
		Email: EmailConfig{
			SMTPHost: "localhost",
			SMTPPort: viper.GetString("MAILHOG_SMTP_PORT"),
			Username: "",
			Password: "",
			From:     "noreply@medflow.local",
		},
	}

	return cfg, nil
}
