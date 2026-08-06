// Package queue оборачивает asynq.Client, чтобы сервисный слой (см.
// internal/service.TaskEnqueuer) не зависел напрямую от asynq - по тому же
// принципу изоляции, что и internal/pkg/storage для MinIO.
package queue

import "github.com/hibiken/asynq"

// QueueCards - имя очереди asynq для задач генерации карточек, должно
// совпадать с ключом в asynq.Config.Queues в cmd/worker/main.go.
const QueueCards = "cards"

type Client struct {
	asynq *asynq.Client
}

func New(redisAddr string) *Client {
	return &Client{asynq: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (c *Client) Close() error {
	return c.asynq.Close()
}

// Enqueue ставит задачу typename с телом payload в очередь QueueCards.
func (c *Client) Enqueue(typename string, payload []byte) error {
	_, err := c.asynq.Enqueue(asynq.NewTask(typename, payload), asynq.Queue(QueueCards))
	return err
}
