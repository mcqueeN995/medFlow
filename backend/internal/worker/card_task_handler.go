// Package worker содержит asynq-обработчики фоновых задач (см.
// cmd/worker/main.go). Бизнес-логика конвейера живёт в internal/service -
// этот пакет только распаковывает payload и логирует исход.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/medflow/backend/internal/service"
)

type CardTaskHandler struct {
	cardService *service.CardService
}

func NewCardTaskHandler(cardService *service.CardService) *CardTaskHandler {
	return &CardTaskHandler{cardService: cardService}
}

// HandleGenerate - обработчик asynq-задачи service.TaskTypeGenerateCards.
// Ошибка, возвращённая отсюда, заставляет asynq повторить задачу - поэтому
// CardService.ProcessTask сам фиксирует бизнес-сбои (невалидный PDF, LLM
// недоступен) в статусе задачи и возвращает nil; ошибка наверх пробрасывается
// только когда сама попытка это зафиксировать не удалась (стоит повторить).
func (h *CardTaskHandler) HandleGenerate(ctx context.Context, task *asynq.Task) error {
	var payload service.GenerateCardsPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal generate cards payload: %w", err)
	}

	taskID, err := uuid.Parse(payload.TaskID)
	if err != nil {
		return fmt.Errorf("invalid task_id in payload: %w", err)
	}

	if err := h.cardService.ProcessTask(ctx, taskID); err != nil {
		slog.Error("card task processing failed", "task_id", taskID, "error", err)
		return err
	}
	return nil
}
