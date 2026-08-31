package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"pack_mate/internal/config"
)

const deepseekMaxErrorBodyLength = 300

type deepseekChatRequest struct {
	Model       string                `json:"model"`
	Messages    []deepseekChatMessage `json:"messages"`
	Temperature float64               `json:"temperature"`
}

type deepseekChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekChatResponse struct {
	Choices []deepseekChatChoice `json:"choices"`
	Error   *deepseekError       `json:"error,omitempty"`
}

type deepseekChatChoice struct {
	Message deepseekChatMessage `json:"message"`
}

type deepseekError struct {
	Message string `json:"message"`
}

// ChatMessage defines one chat message for an LLM.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatCompletionInput defines a generic chat completion request.
type ChatCompletionInput struct {
	Messages    []ChatMessage
	Temperature float64
}

// ChatClient completes chat requests and returns assistant content.
type ChatClient interface {
	CompleteChat(ctx context.Context, input ChatCompletionInput) (string, error)
}

// DeepSeekChatClient completes chat requests with DeepSeek.
type DeepSeekChatClient struct {
	config config.DeepseekConfig
	client *http.Client
}

// NewDeepSeekChatClient creates a DeepSeek chat client.
func NewDeepSeekChatClient(cfg config.DeepseekConfig) *DeepSeekChatClient {
	return &DeepSeekChatClient{
		config: cfg,
		client: http.DefaultClient,
	}
}

// CompleteChat calls DeepSeek chat completions and returns assistant content.
func (c *DeepSeekChatClient) CompleteChat(ctx context.Context, input ChatCompletionInput) (string, error) {
	reqBody := deepseekChatRequest{
		Model:       c.config.Model,
		Messages:    make([]deepseekChatMessage, 0, len(input.Messages)),
		Temperature: input.Temperature,
	}
	for _, message := range input.Messages {
		reqBody.Messages = append(reqBody.Messages, deepseekChatMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal deepseek request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(c.config.BaseURL), bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create deepseek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.config.APIKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call deepseek: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read deepseek response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		message := parseDeepSeekErrorMessage(body)
		if message == "" {
			message = truncateDeepSeekErrorBody(strings.TrimSpace(string(body)))
		}
		return "", fmt.Errorf("deepseek request failed: %s", message)
	}

	var parsed deepseekChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode deepseek response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("deepseek response did not contain choices")
	}

	return parsed.Choices[0].Message.Content, nil
}

func parseDeepSeekErrorMessage(body []byte) string {
	var parsed deepseekChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Error == nil {
		return ""
	}
	return strings.TrimSpace(parsed.Error.Message)
}

func truncateDeepSeekErrorBody(body string) string {
	if len(body) <= deepseekMaxErrorBodyLength {
		return body
	}
	return body[:deepseekMaxErrorBodyLength]
}
