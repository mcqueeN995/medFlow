package config

import (
	"errors"
	"fmt"
	"io/fs"
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
	LLM      LLMConfig
	Ollama   OllamaConfig
	VAPID    VAPIDConfig
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
	// Endpoint - адрес S3/MinIO внутри docker-сети (например, minio:9000),
	// используется бэкендом для записи файлов (PutObject).
	Endpoint string
	// PublicEndpoint - адрес, по которому S3/MinIO реально достижим для
	// браузера клиента (в dev - localhost:9000, порт MinIO проброшен на хост;
	// в проде - домен/CDN перед MinIO). Presigned-ссылки (download/upload)
	// подписываются под этот хост - иначе ссылка с internal-хостом minio:9000
	// была бы нерабочей для внешнего клиента, у которого такого DNS-имени нет.
	PublicEndpoint string
	AccessKey      string
	SecretKey      string
	BucketName     string
	Region         string
	UseSSL         bool
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

// LLMConfig — облачный провайдер генерации карточек (прод по умолчанию).
type LLMConfig struct {
	Provider       string
	APIKey         string
	Model          string
	EmbeddingModel string
}

// OllamaConfig — локальный провайдер для dev-окружения и приватной обработки.
type OllamaConfig struct {
	Host            string
	GenerationModel string
	EmbeddingModel  string
}

// VAPIDConfig — ключевая пара для подписи Web Push уведомлений.
type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// withDefault возвращает значение переменной окружения или def, если она не задана.
// Используется для хостов сервисов: локально (вне Docker) они слушают на localhost
// через проброшенные порты, а в docker-compose приходят как имена сервисов
// (POSTGRES_HOST=postgres и т.д., см. infra/docker-compose.yml).
func withDefault(key, def string) string {
	if v := viper.GetString(key); v != "" {
		return v
	}
	return def
}

func Load() (*Config, error) {
	// ../.env удобен для локального запуска `go run` из backend/, но в контейнере
	// такого файла нет — переменные приходят напрямую через env_file/environment
	// в docker-compose. Отсутствие файла не должно быть фатальной ошибкой.
	viper.SetConfigFile("../.env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// SetConfigFile с явным путём при отсутствии файла возвращает "голую" fs-ошибку
		// (не viper.ConfigFileNotFoundError — тот только для поиска по AddConfigPath).
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
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
			Host:     withDefault("POSTGRES_HOST", "localhost"),
			Port:     viper.GetString("POSTGRES_PORT"),
		},
		Redis: RedisConfig{
			Host: withDefault("REDIS_HOST", "localhost"),
			Port: viper.GetString("REDIS_PORT"),
		},
		S3: S3Config{
			Endpoint:       withDefault("S3_ENDPOINT", fmt.Sprintf("localhost:%s", viper.GetString("MINIO_API_PORT"))),
			PublicEndpoint: withDefault("S3_PUBLIC_ENDPOINT", fmt.Sprintf("localhost:%s", viper.GetString("MINIO_API_PORT"))),
			AccessKey:      viper.GetString("MINIO_ROOT_USER"),
			SecretKey:      viper.GetString("MINIO_ROOT_PASSWORD"),
			BucketName:     viper.GetString("S3_BUCKET_NAME"),
			Region:         viper.GetString("S3_REGION"),
			UseSSL:         false,
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpire:  accessExpire,
			RefreshExpire: refreshExpire,
		},
		Email: EmailConfig{
			SMTPHost: withDefault("SMTP_HOST", "localhost"),
			SMTPPort: viper.GetString("MAILHOG_SMTP_PORT"),
			Username: "",
			Password: "",
			From:     "noreply@medflow.local",
		},
		LLM: LLMConfig{
			Provider:       viper.GetString("LLM_PROVIDER"),
			APIKey:         viper.GetString("LLM_API_KEY"),
			Model:          viper.GetString("LLM_MODEL"),
			EmbeddingModel: viper.GetString("LLM_EMBEDDING_MODEL"),
		},
		Ollama: OllamaConfig{
			Host:            withDefault("OLLAMA_HOST", "http://localhost:11434"),
			GenerationModel: withDefault("OLLAMA_GENERATION_MODEL", "qwen2.5:7b-instruct"),
			EmbeddingModel:  withDefault("OLLAMA_EMBEDDING_MODEL", "bge-m3"),
		},
		VAPID: VAPIDConfig{
			PublicKey:  viper.GetString("VAPID_PUBLIC_KEY"),
			PrivateKey: viper.GetString("VAPID_PRIVATE_KEY"),
			Subject:    withDefault("VAPID_SUBJECT", "mailto:admin@medflow.local"),
		},
	}

	return cfg, nil
}
