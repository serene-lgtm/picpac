package agent

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

type dashscopeRequest struct {
	Model      string              `json:"model"`
	Input      dashscopeInput      `json:"input"`
	Parameters dashscopeParameters `json:"parameters"`
}

type dashscopeInput struct {
	Messages []dashscopeMessage `json:"messages"`
}

type dashscopeMessage struct {
	Role    string             `json:"role"`
	Content []dashscopeContent `json:"content"`
}

type dashscopeContent struct {
	Image string `json:"image,omitempty"`
	Text  string `json:"text,omitempty"`
}

type dashscopeParameters struct {
	N              int    `json:"n"`
	NegativePrompt string `json:"negative_prompt"`
	PromptExtend   bool   `json:"prompt_extend"`
	Watermark      bool   `json:"watermark"`
	Size           string `json:"size"`
}

type dashscopeResponse struct {
	RequestID string          `json:"request_id"`
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Output    dashscopeOutput `json:"output"`
}

type dashscopeOutput struct {
	Choices []dashscopeChoice `json:"choices"`
}

type dashscopeChoice struct {
	Message dashscopeAssistantMessage `json:"message"`
}

type dashscopeAssistantMessage struct {
	Content []dashscopeContent `json:"content"`
}

// ImageProcessingAgent defines image rendering behavior.
type ImageProcessingAgent interface {
	RenderImage(ctx context.Context, originalImageURL string, prompt string) (string, error)
}

// QwenImageAgent renders images with DashScope.
type QwenImageAgent struct {
	config config.DashscopeConfig
	client *http.Client
}

// NewQwenImageAgent creates a Qwen-backed image processing agent.
func NewQwenImageAgent(cfg config.DashscopeConfig) *QwenImageAgent {
	return &QwenImageAgent{
		config: cfg,
		client: http.DefaultClient,
	}
}

// RenderImage calls DashScope and returns the rendered image URL.
func (a *QwenImageAgent) RenderImage(ctx context.Context, originalImageURL string, prompt string) (string, error) {
	reqBody := dashscopeRequest{
		Model: a.config.ImageModel,
		Input: dashscopeInput{
			Messages: []dashscopeMessage{
				{
					Role: "user",
					Content: []dashscopeContent{
						{Image: originalImageURL},
						{Text: prompt},
					},
				},
			},
		},
		Parameters: dashscopeParameters{
			N:              1,
			NegativePrompt: " ",
			PromptExtend:   true,
			Watermark:      false,
			Size:           "768*1024",
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal dashscope request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create dashscope request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(a.config.APIKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("call dashscope: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read dashscope response: %w", err)
	}

	var parsed dashscopeResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decode dashscope response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		if strings.TrimSpace(parsed.Message) != "" {
			return "", fmt.Errorf("dashscope request failed: %s", strings.TrimSpace(parsed.Message))
		}
		return "", fmt.Errorf("dashscope request failed: %s", strings.TrimSpace(string(body)))
	}
	if parsed.Code != "" {
		return "", fmt.Errorf("dashscope request failed: %s", strings.TrimSpace(parsed.Message))
	}

	for _, choice := range parsed.Output.Choices {
		for _, content := range choice.Message.Content {
			if strings.TrimSpace(content.Image) != "" {
				return strings.TrimSpace(content.Image), nil
			}
		}
	}

	return "", fmt.Errorf("dashscope response did not contain generated image url")
}
