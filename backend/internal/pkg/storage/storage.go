// Package storage оборачивает S3-совместимый объектный клиент (в dev/проде
// это MinIO, см. infra/docker-compose.yml) для загрузки файлов и выдачи
// presigned-ссылок на скачивание.
package storage

import (
	"context"
	"io"
	"time"

	"github.com/medflow/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc        *minio.Client // internal endpoint (docker-сеть) - для записи файлов
	presignMc *minio.Client // публичный endpoint - для presigned-ссылок клиенту
	bucket    string
}

func New(cfg config.S3Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	presignMc := mc
	if cfg.PublicEndpoint != cfg.Endpoint {
		presignMc, err = minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
			Region: cfg.Region,
		})
		if err != nil {
			return nil, err
		}
	}

	return &Client{mc: mc, presignMc: presignMc, bucket: cfg.BucketName}, nil
}

func (c *Client) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *Client) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.presignMc.PresignedGetObject(ctx, c.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// Get читает объект целиком на сервере (в отличие от PresignedGetURL, не
// редиректит браузер, а отдаёт байты самому backend/worker - нужно для
// парсинга PDF в конвейере ИИ-карточек, см. internal/pkg/pdf).
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
}

// Remove удаляет объект. Используется, чтобы стереть временный PDF
// пользовательской загрузки сразу после обработки задачи на карточки.
func (c *Client) Remove(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}
