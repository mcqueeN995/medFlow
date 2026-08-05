package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// cloudBaseURLs — базовые URL облачных провайдеров, совместимых с OpenAI
// chat-completions API. Используется и New() (валидация имени провайдера),
// и NewOpenAICompatProvider.
var cloudBaseURLs = map[string]string{
	"deepseek":   "https://api.deepseek.com/v1",
	"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"openrouter": "https://openrouter.ai/api/v1",
}

// OpenAICompatProvider — облачный провайдер (DeepSeek/Qwen/OpenRouter),
// говорящий по общему OpenAI chat-completions протоколу. Прод-дефолт: см.
// решение "прод использует облачный LLM, Ollama — только для разработки".
//
// Embed не реализован по-настоящему: у DeepSeek/Qwen/OpenRouter нет единого
// рабочего контракта на эмбеддинги (у OpenRouter эмбеддингов нет вовсе), см.
// ErrEmbedNotSupported.
type OpenAICompatProvider struct {
	baseURL        string
	apiKey         string
	model          string
	embeddingModel string
	httpClient     *http.Client
}

func NewOpenAICompatProvider(baseURL, apiKey, model, embeddingModel string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		baseURL:        baseURL,
		apiKey:         apiKey,
		model:          model,
		embeddingModel: embeddingModel,
		httpClient:     &http.Client{Timeout: defaultHTTPTimeout},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (p *OpenAICompatProvider) Generate(ctx context.Context, prompt string) (string, error) {
	var result string
	err := withRetry(ctx, func() error {
		var out chatCompletionsResponse
		req := chatCompletionsRequest{
			Model:    p.model,
			Messages: []chatMessage{{Role: "user", Content: prompt}},
			Stream:   false,
		}
		if err := p.post(ctx, "/chat/completions", req, &out); err != nil {
			return err
		}
		if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
			return fmt.Errorf("%w: empty choices", ErrInvalidResponse)
		}
		result = out.Choices[0].Message.Content
		return nil
	})
	if err != nil {
		return "", err
	}
	return result, nil
}

// Embed намеренно не выполняет запрос — см. doc-комментарий типа и ErrEmbedNotSupported.
func (p *OpenAICompatProvider) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, ErrEmbedNotSupported
}

// post отправляет reqBody как JSON на p.baseURL+path и декодирует ответ в out.
// Сетевые ошибки и 5xx помечаются markRetryable — их обрабатывает withRetry в
// Generate. Остальные не-2xx (401/400/429 и т.п.) возвращаются как есть:
// retry бессмыслен для неверного ключа или некорректного запроса.
func (p *OpenAICompatProvider) post(ctx context.Context, path string, reqBody, out any) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		// Сетевая ошибка — транзиентная, withRetry сам обернёт в ErrProviderUnavailable
		// после исчерпания попыток.
		return markRetryable(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return markRetryable(fmt.Errorf("cloud provider status %d", resp.StatusCode))
	}
	if resp.StatusCode >= 300 {
		// 4xx (неверный ключ, некорректный запрос, rate limit) — постоянный сбой,
		// retry бессмыслен. Намеренно не оборачиваем в ErrProviderUnavailable.
		return fmt.Errorf("cloud provider status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return nil
}
