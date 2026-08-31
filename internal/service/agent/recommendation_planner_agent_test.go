package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pack_mate/internal/service/llm"
)

type fakeChatClient struct {
	content string
	err     error
	input   llm.ChatCompletionInput
}

func (c *fakeChatClient) CompleteChat(_ context.Context, input llm.ChatCompletionInput) (string, error) {
	c.input = input
	if c.err != nil {
		return "", c.err
	}
	return c.content, nil
}

func TestRecommendationPlannerAgentParsesRecommendationRefs(t *testing.T) {
	t.Parallel()

	chat := &fakeChatClient{content: "```json\n{\"refs\":[\"i1\",\"i3\"]}\n```"}
	recommender := NewRecommendationPlannerAgent(chat)

	result, err := recommender.Recommend(context.Background(), RecommendationInput{
		Scenario: "pack_item_recommendation",
		Subject: RecommendationSubject{
			Type: "pack",
			Name: "日本出差",
		},
		Limit: 15,
		Candidates: []RecommendationCandidate{
			{Ref: "i1", Name: "护照"},
			{Ref: "i3", Name: "充电器"},
		},
	})
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}
	if len(result.Refs) != 2 || result.Refs[0] != "i1" || result.Refs[1] != "i3" {
		t.Fatalf("unexpected refs: %+v", result.Refs)
	}
	if len(chat.input.Messages) != 2 {
		t.Fatalf("expected 2 chat messages, got %d", len(chat.input.Messages))
	}
	if !strings.Contains(chat.input.Messages[1].Content, "pack_item_recommendation") {
		t.Fatalf("expected scenario in prompt, got %s", chat.input.Messages[1].Content)
	}
	if !strings.Contains(chat.input.Messages[0].Content, "出行目的") ||
		!strings.Contains(chat.input.Messages[0].Content, "主要活动") ||
		!strings.Contains(chat.input.Messages[0].Content, "环境气候") ||
		!strings.Contains(chat.input.Messages[0].Content, "不适用场景") {
		t.Fatalf("expected scene dimensions in system prompt, got %s", chat.input.Messages[0].Content)
	}
	if !strings.Contains(chat.input.Messages[0].Content, "商务出差") ||
		!strings.Contains(chat.input.Messages[0].Content, "休闲度假") ||
		!strings.Contains(chat.input.Messages[0].Content, "会议") ||
		!strings.Contains(chat.input.Messages[0].Content, "徒步") {
		t.Fatalf("expected reference items in system prompt, got %s", chat.input.Messages[0].Content)
	}
	if !strings.Contains(chat.input.Messages[0].Content, "普通城市环境") ||
		!strings.Contains(chat.input.Messages[0].Content, "高海拔") ||
		!strings.Contains(chat.input.Messages[0].Content, "不要推荐冲锋衣") {
		t.Fatalf("expected climate and exclusion rules in system prompt, got %s", chat.input.Messages[0].Content)
	}
}

func TestRecommendationPlannerAgentWrapsChatError(t *testing.T) {
	t.Parallel()

	recommender := NewRecommendationPlannerAgent(&fakeChatClient{err: errors.New("llm down")})

	_, err := recommender.Recommend(context.Background(), RecommendationInput{Scenario: "pack_item_recommendation", Limit: 15})
	if err == nil || !strings.Contains(err.Error(), "complete recommendation chat") {
		t.Fatalf("expected chat completion failure, got %v", err)
	}
}

func TestRecommendationPlannerAgentReturnsErrorForInvalidRecommendationJSON(t *testing.T) {
	t.Parallel()

	recommender := NewRecommendationPlannerAgent(&fakeChatClient{content: "not json"})

	_, err := recommender.Recommend(context.Background(), RecommendationInput{Scenario: "pack_item_recommendation", Limit: 15})
	if err == nil || !strings.Contains(err.Error(), "decode recommendation planner response") {
		t.Fatalf("expected recommendation decode failure, got %v", err)
	}
}
