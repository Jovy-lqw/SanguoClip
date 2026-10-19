package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Jovy-lqw/SanguoClip/internal/domain"
)

type Client struct {
	httpClient *http.Client
}

type Suggestion struct {
	Segments []domain.Segment `json:"segments"`
	Summary  string           `json:"summary"`
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 90 * time.Second}}
}

func (c *Client) Suggest(ctx context.Context, config domain.ModelConfig, style domain.Style, duration float64, notes string) (Suggestion, error) {
	if config.Model == "" {
		return Suggestion{}, errors.New("请先在模型配置中填写模型名称")
	}
	baseURL, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" {
		return Suggestion{}, errors.New("模型接口地址无效")
	}
	apiKey := os.Getenv(config.APIKeyEnv)
	if config.APIKeyEnv != "" && apiKey == "" {
		return Suggestion{}, fmt.Errorf("环境变量 %s 尚未设置", config.APIKeyEnv)
	}

	systemPrompt := `你是游戏视频剪辑师。根据素材时长、创作备注和风格要求输出严格 JSON，不要输出 Markdown。格式为 {"segments":[{"start":数字,"end":数字,"label":"说明"}],"summary":"简述"}。片段必须按时间升序、互不重叠，start >= 0，end > start，end 不得超过素材时长。只根据用户提供的信息决策，不得声称看过视频。`
	userPrompt := fmt.Sprintf("素材时长：%.3f 秒\n风格名称：%s\n风格要求：%s\n创作备注：%s",
		duration, style.Name, style.Prompt, strings.TrimSpace(notes))

	requestBody := map[string]any{
		"model": config.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": config.Temperature,
	}
	body, _ := json.Marshal(requestBody)
	endpoint := strings.TrimRight(baseURL.String(), "/") + "/chat/completions"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Suggestion{}, fmt.Errorf("创建模型请求失败: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Suggestion{}, fmt.Errorf("请求模型失败: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return Suggestion{}, fmt.Errorf("读取模型响应失败: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Suggestion{}, fmt.Errorf("模型接口返回 %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &completion); err != nil || len(completion.Choices) == 0 {
		return Suggestion{}, errors.New("模型响应格式不兼容 OpenAI Chat Completions")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var suggestion Suggestion
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &suggestion); err != nil {
		return Suggestion{}, fmt.Errorf("模型未返回有效 EDL JSON: %w", err)
	}
	if err := validateSegments(suggestion.Segments, duration); err != nil {
		return Suggestion{}, fmt.Errorf("模型返回的片段无效: %w", err)
	}
	return suggestion, nil
}

func validateSegments(segments []domain.Segment, duration float64) error {
	if len(segments) == 0 {
		return errors.New("没有片段")
	}
	var previousEnd float64
	for index, segment := range segments {
		if segment.Start < 0 || segment.End <= segment.Start {
			return fmt.Errorf("第 %d 个片段时间范围错误", index+1)
		}
		if duration > 0 && segment.End > duration+0.001 {
			return fmt.Errorf("第 %d 个片段超过素材时长", index+1)
		}
		if index > 0 && segment.Start < previousEnd {
			return fmt.Errorf("第 %d 个片段与前一片段重叠", index+1)
		}
		previousEnd = segment.End
	}
	return nil
}
