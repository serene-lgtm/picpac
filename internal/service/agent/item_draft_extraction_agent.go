package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pack_mate/internal/service/llm"
)

const itemDraftExtractionSystemPrompt = `你是一个移动端个人物品管理应用的物品草稿提取 Agent。你的任务是从用户输入中提取用户明确想添加的物品，并为每个物品选择一个最合适的 category_key。

规则：
- 只能提取用户明确提到的物品，不要补充建议物品。
- 不要把明显属于一个完整物品名的短语过度拆分，例如“手机充电线”应保留为一个物品，除非用户明确分别列出“手机、充电线”。
- 去掉“请帮我添加”“帮我加一下”等指令性文字，只保留物品名称。
- 物品名称应简洁、自然，适合展示给用户。
- category_key 必须从 categories 中选择；不确定时使用 other。
- 最终只返回合法 JSON，格式必须是 {"items":[{"name":"手机","category_key":"electronics"}]}。不要解释，不要输出 markdown。`

type itemDraftExtractionContent struct {
	Items []ItemDraft `json:"items"`
}

// ItemDraftCategory defines one category option sent to the extraction agent.
type ItemDraftCategory struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ItemDraft defines one extracted item draft.
type ItemDraft struct {
	Name        string `json:"name"`
	CategoryKey string `json:"category_key"`
}

// ItemDraftExtractionInput defines input for extracting item drafts.
type ItemDraftExtractionInput struct {
	Text       string              `json:"text"`
	Categories []ItemDraftCategory `json:"categories"`
}

// ItemDraftExtractionAgent extracts item drafts from natural language.
type ItemDraftExtractionAgent interface {
	ExtractItemDrafts(ctx context.Context, input ItemDraftExtractionInput) ([]ItemDraft, error)
}

// ChatItemDraftExtractionAgent extracts item drafts with a chat LLM.
type ChatItemDraftExtractionAgent struct {
	chat llm.ChatClient
}

// NewChatItemDraftExtractionAgent creates a chat-backed item draft extraction agent.
func NewChatItemDraftExtractionAgent(chat llm.ChatClient) *ChatItemDraftExtractionAgent {
	return &ChatItemDraftExtractionAgent{chat: chat}
}

// ExtractItemDrafts extracts item drafts from natural language.
func (a *ChatItemDraftExtractionAgent) ExtractItemDrafts(ctx context.Context, input ItemDraftExtractionInput) ([]ItemDraft, error) {
	contextPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal item draft extraction context: %w", err)
	}

	content, err := a.chat.CompleteChat(ctx, llm.ChatCompletionInput{
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: itemDraftExtractionSystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请从以下用户输入中提取待创建物品草稿，并为每个物品选择 category_key。上下文 JSON：%s", string(contextPayload)),
			},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("complete item draft extraction chat: %w", err)
	}

	var parsed itemDraftExtractionContent
	if err := json.Unmarshal([]byte(normalizeAgentJSON(content)), &parsed); err != nil {
		return nil, fmt.Errorf("decode item draft extraction response: %w", err)
	}

	return parsed.Items, nil
}

func normalizeAgentJSON(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}
