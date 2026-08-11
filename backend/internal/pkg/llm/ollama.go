package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaProvider — локальный LLM-бэкенд через нативный API Ollama.
// Retry намеренно не применяется: Ollama живёт локально (dev-окружение или
// свой сервер), а требование на экспоненциальный backoff в ТЗ адресовано
// облачному провайдеру как более "флаки" сети (см. openai_compat.go).
type OllamaProvider struct {
	host            string
	generationModel string
	embeddingModel  string
	httpClient      *http.Client
}

// ollamaHTTPTimeout — таймаут отдельный от defaultHTTPTimeout: облачные API
// отвечают за секунды, а локальная генерация 7B-моделью без GPU (dev-Mac
// без видеокарты) на полный промпт с карточками может занимать несколько
// минут - 60s (общий таймаут для облака) там регулярно ловит "context
// deadline exceeded" ещё до того, как модель успела ответить.
const ollamaHTTPTimeout = 5 * time.Minute

func NewOllamaProvider(host, generationModel, embeddingModel string) *OllamaProvider {
	return &OllamaProvider{
		host:            host,
		generationModel: generationModel,
		embeddingModel:  embeddingModel,
		httpClient:      &http.Client{Timeout: ollamaHTTPTimeout},
	}
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

type ollamaEmbeddingsRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingsResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	var out ollamaGenerateResponse
	req := ollamaGenerateRequest{Model: p.generationModel, Prompt: prompt, Stream: false}
	if err := p.post(ctx, "/api/generate", req, &out); err != nil {
		return "", err
	}
	if out.Response == "" {
		return "", fmt.Errorf("%w: empty response field", ErrInvalidResponse)
	}
	return out.Response, nil
}

func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	var out ollamaEmbeddingsResponse
	req := ollamaEmbeddingsRequest{Model: p.embeddingModel, Prompt: text}
	if err := p.post(ctx, "/api/embeddings", req, &out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("%w: empty embedding field", ErrInvalidResponse)
	}
	return out.Embedding, nil
}

// post отправляет reqBody как JSON на p.host+path и декодирует ответ в out.
func (p *OllamaProvider) post(ctx context.Context, path string, reqBody, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.host+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%w: ollama status %d", ErrProviderUnavailable, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return nil
}
