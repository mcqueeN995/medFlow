package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudBaseURLs(t *testing.T) {
	want := map[string]string{
		"deepseek":   "https://api.deepseek.com/v1",
		"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
		"openrouter": "https://openrouter.ai/api/v1",
	}
	if len(cloudBaseURLs) != len(want) {
		t.Fatalf("cloudBaseURLs has %d entries, want %d", len(cloudBaseURLs), len(want))
	}
	for key, url := range want {
		if cloudBaseURLs[key] != url {
			t.Errorf("cloudBaseURLs[%q] = %q, want %q", key, cloudBaseURLs[key], url)
		}
	}
}

func TestOpenAICompatProvider_Generate_Success(t *testing.T) {
	var gotAuth string
	var gotBody chatCompletionsRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		resp := chatCompletionsResponse{}
		resp.Choices = []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "hi there"}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "sk-secret", "deepseek-chat", "unused")
	got, err := p.Generate(context.Background(), "say hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hi there" {
		t.Fatalf("got %q", got)
	}
	if gotAuth != "Bearer sk-secret" {
		t.Fatalf("Authorization header = %q", gotAuth)
	}
	if gotBody.Model != "deepseek-chat" || len(gotBody.Messages) != 1 || gotBody.Messages[0].Content != "say hi" || gotBody.Stream {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
}

func TestOpenAICompatProvider_Generate_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatCompletionsResponse{})
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "key", "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestOpenAICompatProvider_Generate_RetriesOn5xxThenSucceeds(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := chatCompletionsResponse{}
		resp.Choices = []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Content: "ok"}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "key", "model", "embed-model")
	got, err := p.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestOpenAICompatProvider_Generate_ExhaustsRetries(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "key", "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if want := len(retryDelays) + 1; calls != want {
		t.Fatalf("expected %d calls, got %d", want, calls)
	}
}

func TestOpenAICompatProvider_Generate_4xxNotRetried(t *testing.T) {
	withShortRetryDelays(t)

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := NewOpenAICompatProvider(server.URL, "bad-key", "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("4xx should not be classified as ErrProviderUnavailable, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (4xx not retried), got %d", calls)
	}
}

func TestOpenAICompatProvider_Generate_NetworkErrorRetriedThenFails(t *testing.T) {
	withShortRetryDelays(t)

	// Порт, на котором заведомо никто не слушает — имитация сетевой ошибки.
	p := NewOpenAICompatProvider("http://127.0.0.1:1", "key", "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable after exhausting retries, got %v", err)
	}
}

func TestOpenAICompatProvider_Embed_NotSupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Embed must not perform any HTTP request, got %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	for provider := range cloudBaseURLs {
		t.Run(provider, func(t *testing.T) {
			p := NewOpenAICompatProvider(server.URL, "key", "model", "embed-model")
			_, err := p.Embed(context.Background(), "some text")
			if !errors.Is(err, ErrEmbedNotSupported) {
				t.Fatalf("expected ErrEmbedNotSupported, got %v", err)
			}
		})
	}
}
