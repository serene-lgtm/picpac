package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"pack_mate/internal/service/llm"
)

type recommendationPlannerContent struct {
	Refs []string `json:"refs"`
}

const recommendationPlannerSystemPrompt = `你是一个移动端打包应用的通用推荐规划 Agent。你只能从候选项 candidates 中选择 ref，不能创造新的候选项。

推荐前必须先基于 subject 的名称和描述判断四个要素：出行目的、主要活动、环境气候、不适用场景。

出行目的参考项：商务出差、学习培训、休闲度假、城市旅行、家庭亲子、探亲返乡、户外徒步、户外露营、自驾旅行、高原旅行、冰雪运动、水上运动、运动赛事、摄影旅行、社交活动、医疗陪护、搬家整理、日常短途。

主要活动参考项：飞行、高铁或火车、开车自驾、酒店住宿、会议、客户拜访、培训学习、面试、通勤、城市散步、购物、美食探店、海边游玩、游泳、浮潜、潜水、出海、徒步、登山、露营、滑雪、骑行、跑步比赛、摄影、亲子照看、探亲拜访、婚礼或晚宴、医院就诊或陪诊、搬运整理。

环境气候参考项：普通城市环境、炎热、寒冷、温差大、晴晒、潮湿、多雨、大风、降雪、干燥、高海拔、海边、野外。

只推荐与出行目的、主要活动、环境气候强相关的候选项。衣物、防护、药品类候选项必须匹配主要活动或环境气候，不能只因为“旅行可能用到”就推荐。特殊场景物品不能泛化推荐；如果当前场景没有寒冷、多雨、大风、降雪、户外徒步、登山、露营、滑雪、高海拔等信息，不要推荐冲锋衣、保暖衣物、红景天等强场景物品。如果候选项明显属于不适用场景，必须排除；如果不确定，宁可少推荐。

最终只返回合法 JSON，格式必须是 {"refs":["i1"]}。不要解释，不要输出 markdown。`

// RecommendationSubject defines the business object that needs recommendations.
type RecommendationSubject struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// RecommendationCandidate defines one selectable candidate sent to the agent.
type RecommendationCandidate struct {
	Ref         string            `json:"ref"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// RecommendationInput defines generic recommendation input.
type RecommendationInput struct {
	Scenario   string                    `json:"scenario"`
	Subject    RecommendationSubject     `json:"subject"`
	Limit      int                       `json:"limit"`
	Candidates []RecommendationCandidate `json:"candidates"`
}

// RecommendationResult defines generic recommended candidate refs.
type RecommendationResult struct {
	Refs []string
}

// RecommendationPlanner recommends candidate refs from a provided candidate list.
type RecommendationPlanner interface {
	Recommend(ctx context.Context, input RecommendationInput) (*RecommendationResult, error)
}

// RecommendationPlannerAgent recommends candidates with a chat LLM.
type RecommendationPlannerAgent struct {
	chat llm.ChatClient
}

// NewRecommendationPlannerAgent creates a recommendation planner agent.
func NewRecommendationPlannerAgent(chat llm.ChatClient) *RecommendationPlannerAgent {
	return &RecommendationPlannerAgent{chat: chat}
}

// Recommend returns recommended candidate refs for a recommendation context.
func (a *RecommendationPlannerAgent) Recommend(ctx context.Context, input RecommendationInput) (*RecommendationResult, error) {
	contextPayload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal recommendation context: %w", err)
	}

	content, err := a.chat.CompleteChat(ctx, llm.ChatCompletionInput{
		Messages: []llm.ChatMessage{
			{
				Role:    "system",
				Content: recommendationPlannerSystemPrompt,
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("请从候选项中选择最多 %d 个最适合当前推荐场景的 ref。请在内部先分析 subject 对应的出行目的、主要活动、环境气候和不适用场景，然后再选择候选项。推荐上下文 JSON：%s", input.Limit, string(contextPayload)),
			},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("complete recommendation chat: %w", err)
	}

	var recommendation recommendationPlannerContent
	if err := json.Unmarshal([]byte(normalizeAgentJSON(content)), &recommendation); err != nil {
		return nil, fmt.Errorf("decode recommendation planner response: %w", err)
	}

	return &RecommendationResult{Refs: recommendation.Refs}, nil
}
