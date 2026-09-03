package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pack_mate/internal/service/llm"
)

type fakeItemDraftChatClient struct {
	content string
	err     error
	input   llm.ChatCompletionInput
}

func (c *fakeItemDraftChatClient) CompleteChat(_ context.Context, input llm.ChatCompletionInput) (string, error) {
	c.input = input
	if c.err != nil {
		return "", c.err
	}
	return c.content, nil
}

func TestChatItemDraftExtractionAgentExtractsDrafts(t *testing.T) {
	t.Parallel()

	chat := &fakeItemDraftChatClient{content: "```json\n{\"items\":[{\"name\":\"手机\",\"category_key\":\"electronics\"}]}\n```"}
	extractor := NewChatItemDraftExtractionAgent(chat)

	drafts, err := extractor.ExtractItemDrafts(context.Background(), ItemDraftExtractionInput{
		Text: "请帮我添加手机",
		Categories: []ItemDraftCategory{
			{Key: "electronics", Name: "电子设备"},
			{Key: "other", Name: "其他"},
		},
	})
	if err != nil {
		t.Fatalf("ExtractItemDrafts returned error: %v", err)
	}
	if len(drafts) != 1 || drafts[0].Name != "手机" || drafts[0].CategoryKey != "electronics" {
		t.Fatalf("unexpected drafts: %+v", drafts)
	}
	if len(chat.input.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(chat.input.Messages))
	}
	if !strings.Contains(chat.input.Messages[0].Content, "只能提取用户明确提到的物品") {
		t.Fatalf("expected extraction rules in system prompt, got %s", chat.input.Messages[0].Content)
	}
}

func TestChatItemDraftExtractionAgentWrapsChatError(t *testing.T) {
	t.Parallel()

	extractor := NewChatItemDraftExtractionAgent(&fakeItemDraftChatClient{err: errors.New("llm down")})

	_, err := extractor.ExtractItemDrafts(context.Background(), ItemDraftExtractionInput{Text: "手机"})
	if err == nil || !strings.Contains(err.Error(), "complete item draft extraction chat") {
		t.Fatalf("expected chat failure, got %v", err)
	}
}

func TestChatItemDraftExtractionAgentRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	extractor := NewChatItemDraftExtractionAgent(&fakeItemDraftChatClient{content: "not json"})

	_, err := extractor.ExtractItemDrafts(context.Background(), ItemDraftExtractionInput{Text: "手机"})
	if err == nil || !strings.Contains(err.Error(), "decode item draft extraction response") {
		t.Fatalf("expected decode failure, got %v", err)
	}
}
