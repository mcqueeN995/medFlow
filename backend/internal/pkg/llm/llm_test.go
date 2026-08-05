package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/medflow/backend/internal/config"
)

func TestWithRetry_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_RetriesThenSucceeds(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return markRetryable(errors.New("transient"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	err := withRetry(context.Background(), func() error {
		calls++
		return markRetryable(errors.New("always fails"))
	})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if want := len(retryDelays) + 1; calls != want {
		t.Fatalf("expected %d calls, got %d", want, calls)
	}
}

func TestWithRetry_PermanentErrorNotRetried(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	sentinel := errors.New("permanent")
	err := withRetry(context.Background(), func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_ContextCanceledDuringWait(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{time.Hour}
	t.Cleanup(func() { retryDelays = orig })

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	done := make(chan error, 1)
	go func() {
		done <- withRetry(ctx, func() error {
			calls++
			return markRetryable(errors.New("transient"))
		})
	}()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("withRetry did not return promptly after context cancellation")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call before cancellation, got %d", calls)
	}
}

func TestNew_Ollama(t *testing.T) {
	cfg := &config.Config{
		LLM: config.LLMConfig{Provider: "ollama"},
		Ollama: config.OllamaConfig{
			Host:            "http://localhost:11434",
			GenerationModel: "qwen2.5:7b-instruct",
			EmbeddingModel:  "bge-m3",
		},
	}

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ollama, ok := provider.(*OllamaProvider)
	if !ok {
		t.Fatalf("expected *OllamaProvider, got %T", provider)
	}
	if ollama.host != cfg.Ollama.Host || ollama.generationModel != cfg.Ollama.GenerationModel || ollama.embeddingModel != cfg.Ollama.EmbeddingModel {
		t.Fatalf("OllamaProvider fields do not match config: %+v", ollama)
	}
}

func TestNew_CloudProviders(t *testing.T) {
	for key, wantBaseURL := range cloudBaseURLs {
		t.Run(key, func(t *testing.T) {
			cfg := &config.Config{
				LLM: config.LLMConfig{
					Provider:       key,
					APIKey:         "secret",
					Model:          "some-model",
					EmbeddingModel: "some-embedding-model",
				},
			}

			provider, err := New(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			cloud, ok := provider.(*OpenAICompatProvider)
			if !ok {
				t.Fatalf("expected *OpenAICompatProvider, got %T", provider)
			}
			if cloud.baseURL != wantBaseURL {
				t.Fatalf("baseURL = %q, want %q", cloud.baseURL, wantBaseURL)
			}
			if cloud.apiKey != "secret" || cloud.model != "some-model" {
				t.Fatalf("OpenAICompatProvider fields do not match config: %+v", cloud)
			}
		})
	}
}

func TestNew_UnknownProvider(t *testing.T) {
	for _, provider := range []string{"bogus", ""} {
		t.Run(provider, func(t *testing.T) {
			cfg := &config.Config{LLM: config.LLMConfig{Provider: provider}}
			_, err := New(cfg)
			if !errors.Is(err, ErrUnsupportedProvider) {
				t.Fatalf("expected ErrUnsupportedProvider, got %v", err)
			}
		})
	}
}

// withShortRetryDelays подменяет retryDelays на миллисекундные значения на
// время теста, чтобы не ждать реальные 1s/2s/4s/8s.
func withShortRetryDelays(t *testing.T) {
	t.Helper()
	orig := retryDelays
	retryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() { retryDelays = orig })
}
