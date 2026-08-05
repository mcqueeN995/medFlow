// Package llm абстрагирует операции генерации текста и получения эмбеддингов
// над разными LLM-бэкендами для RAG-конвейера ИИ-карточек: локальным Ollama
// (dev-окружение, приватная обработка) и облачными провайдерами, совместимыми
// с OpenAI chat-API — DeepSeek/Qwen/OpenRouter (прод по умолчанию).
package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/medflow/backend/internal/config"
)

var (
	// ErrUnsupportedProvider возвращается New, если cfg.LLM.Provider не
	// соответствует ни одному известному провайдеру.
	ErrUnsupportedProvider = errors.New("llm: unsupported provider")

	// ErrProviderUnavailable возвращается провайдером после исчерпания
	// попыток retry (или сразу — для провайдеров без retry, см. ollama.go).
	ErrProviderUnavailable = errors.New("llm: provider unavailable")

	// ErrInvalidResponse возвращается, если провайдер ответил успешным HTTP-
	// статусом, но тело не удалось разобрать или в нём нет ожидаемых полей.
	ErrInvalidResponse = errors.New("llm: invalid response from provider")

	// ErrEmbedNotSupported возвращается облачным провайдером: единого рабочего
	// контракта на эмбеддинги у DeepSeek/Qwen/OpenRouter нет (у OpenRouter
	// эмбеддингов нет вовсе), а размерность вектора в БД (vector(1024))
	// зафиксирована под bge-m3 через Ollama. Эмбеддинги всегда идут через
	// Ollama, независимо от того, кто генерирует текст карточек.
	ErrEmbedNotSupported = errors.New("llm: embeddings not supported by this provider")
)

// defaultHTTPTimeout — таймаут одной попытки HTTP-запроса к LLM-провайдеру.
const defaultHTTPTimeout = 60 * time.Second

// retryDelays — расписание экспоненциального backoff для транзиентных ошибок
// облачного провайдера (1s, 2s, 4s, 8s). var, а не const — тесты подменяют
// его на короткие задержки.
var retryDelays = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}

// Provider — абстракция над конкретным LLM-бэкендом.
type Provider interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

// retryableError маркирует ошибку как транзиентную — подлежащую retry в withRetry.
// Ошибки, не обёрнутые markRetryable, считаются постоянными и возвращаются немедленно.
type retryableError struct{ err error }

func (r *retryableError) Error() string { return r.err.Error() }
func (r *retryableError) Unwrap() error { return r.err }

// markRetryable оборачивает err как транзиентный. nil проходит насквозь.
func markRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &retryableError{err: err}
}

// withRetry выполняет fn по расписанию retryDelays. Ошибки, не помеченные
// markRetryable, возвращаются немедленно (постоянный сбой — retry бессмыслен).
// После исчерпания retryDelays возвращается ErrProviderUnavailable. Отмена ctx
// во время ожидания прерывает retry и возвращает ctx.Err().
func withRetry(ctx context.Context, fn func() error) error {
	var rErr *retryableError
	for attempt := 0; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		if !errors.As(err, &rErr) {
			return err
		}
		if attempt >= len(retryDelays) {
			return fmt.Errorf("%w: %v", ErrProviderUnavailable, rErr.Unwrap())
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelays[attempt]):
		}
	}
}

// New создаёт Provider согласно cfg.LLM.Provider:
//   - "ollama" — локальный провайдер (см. ollama.go);
//   - "deepseek" / "qwen" / "openrouter" — облачный OpenAI-совместимый провайдер
//     (см. openai_compat.go), базовый URL берётся из cloudBaseURLs.
//
// Неизвестное значение возвращает ErrUnsupportedProvider.
func New(cfg *config.Config) (Provider, error) {
	if cfg.LLM.Provider == "ollama" {
		return NewOllamaProvider(cfg.Ollama.Host, cfg.Ollama.GenerationModel, cfg.Ollama.EmbeddingModel), nil
	}

	baseURL, ok := cloudBaseURLs[cfg.LLM.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedProvider, cfg.LLM.Provider)
	}
	return NewOpenAICompatProvider(baseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.EmbeddingModel), nil
}
