package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaProvider_Generate_Success(t *testing.T) {
	var gotBody ollamaGenerateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/generate" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(ollamaGenerateResponse{Response: "hello from ollama"})
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "qwen2.5:7b-instruct", "bge-m3")
	got, err := p.Generate(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello from ollama" {
		t.Fatalf("got %q", got)
	}
	if gotBody.Model != "qwen2.5:7b-instruct" || gotBody.Prompt != "say hello" || gotBody.Stream != false {
		t.Fatalf("unexpected request body: %+v", gotBody)
	}
}

func TestOllamaProvider_Generate_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaGenerateResponse{Response: ""})
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestOllamaProvider_Generate_ServerErrorNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected ErrProviderUnavailable, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 request (no retry for Ollama), got %d", calls)
	}
}

func TestOllamaProvider_Generate_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "model", "embed-model")
	_, err := p.Generate(context.Background(), "prompt")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestOllamaProvider_Embed_Success(t *testing.T) {
	want := []float32{0.1, 0.2, 0.3}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ollamaEmbeddingsResponse{Embedding: want})
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "model", "bge-m3")
	got, err := p.Embed(context.Background(), "some chunk of text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOllamaProvider_Embed_EmptyEmbedding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaEmbeddingsResponse{Embedding: nil})
	}))
	defer server.Close()

	p := NewOllamaProvider(server.URL, "model", "embed-model")
	_, err := p.Embed(context.Background(), "text")
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestOllamaProvider_Generate_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := NewOllamaProvider(server.URL, "model", "embed-model")
	done := make(chan error, 1)
	go func() {
		_, err := p.Generate(ctx, "prompt")
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error for canceled context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Generate did not return promptly after context cancellation")
	}
}
