package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pack_mate/internal/config"
)

func TestDeepSeekChatClientCompletesChat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %s", r.Header.Get("Authorization"))
		}
		var req deepseekChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "deepseek-chat" {
			t.Fatalf("unexpected model: %s", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Fatalf("unexpected messages: %+v", req.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"refs\":[\"i1\"]}"}}]}`))
	}))
	defer server.Close()

	client := NewDeepSeekChatClient(config.DeepseekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-chat",
		BaseURL: server.URL,
	})
	client.client = server.Client()

	content, err := client.CompleteChat(context.Background(), ChatCompletionInput{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("CompleteChat returned error: %v", err)
	}
	if content != "{\"refs\":[\"i1\"]}" {
		t.Fatalf("unexpected content: %s", content)
	}
}

func TestDeepSeekChatClientReturnsErrorForNonOKResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer server.Close()

	client := NewDeepSeekChatClient(config.DeepseekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-chat",
		BaseURL: server.URL,
	})
	client.client = server.Client()

	_, err := client.CompleteChat(context.Background(), ChatCompletionInput{})
	if err == nil || !strings.Contains(err.Error(), "deepseek request failed") {
		t.Fatalf("expected deepseek request failure, got %v", err)
	}
}

func TestDeepSeekChatClientReturnsErrorForInvalidResponseJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewDeepSeekChatClient(config.DeepseekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-chat",
		BaseURL: server.URL,
	})
	client.client = server.Client()

	_, err := client.CompleteChat(context.Background(), ChatCompletionInput{})
	if err == nil || !strings.Contains(err.Error(), "decode deepseek response") {
		t.Fatalf("expected deepseek decode failure, got %v", err)
	}
}
