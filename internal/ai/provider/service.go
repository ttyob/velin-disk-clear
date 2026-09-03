package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"ai-clear/internal/storage"
)

type Service struct {
	store   *storage.Store
	secrets credentialStore
}

func New(dataDir string) (*Service, error) {
	store, err := storage.Open(dataDir)
	if err != nil {
		return nil, err
	}
	secrets, err := newCredentialStore(dataDir)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, secrets: secrets}, nil
}

func (s *Service) Get() (Config, error) {
	payload, found, err := s.store.Provider("provider-default")
	if err != nil {
		return Config{}, err
	}
	if !found {
		return Config{ID: "provider-default", Name: "Cleaning Agent", Protocol: "openai_compatible", TimeoutSeconds: 60, MaxOutputTokens: 4096}, nil
	}
	var config Config
	if err := json.Unmarshal([]byte(payload), &config); err != nil {
		return Config{}, fmt.Errorf("decode provider config: %w", err)
	}
	_, keyErr := s.secrets.Load(config.ID)
	config.KeySaved = keyErr == nil
	return config, nil
}

func (s *Service) Save(input ConfigInput) (Config, error) {
	normalized, err := validateInput(input)
	if err != nil {
		return Config{}, err
	}
	current, _ := s.Get()
	if current.ID == "" {
		current.ID = "provider-default"
	}
	config := Config{ID: current.ID, Name: normalized.Name, Protocol: "openai_compatible", BaseURL: normalized.BaseURL, Model: normalized.Model, CredentialRef: "dpapi://ai-clear/" + current.ID, TimeoutSeconds: normalized.TimeoutSeconds, MaxOutputTokens: normalized.MaxOutputTokens, Enabled: true}
	if input.APIKey != "" {
		if err := s.secrets.Save(config.ID, input.APIKey); err != nil {
			return Config{}, fmt.Errorf("save API credential: %w", err)
		}
		config.KeySaved = true
	} else if _, err := s.secrets.Load(config.ID); err == nil {
		config.KeySaved = true
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return Config{}, err
	}
	if err := s.store.SaveProvider(config.ID, config.Enabled, string(data)); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (s *Service) Test(ctx context.Context, input ConfigInput) (TestResult, error) {
	config, err := s.Save(input)
	if err != nil {
		return TestResult{}, err
	}
	key := input.APIKey
	if key == "" {
		key, _ = s.secrets.Load(config.ID)
	}
	endpoint := strings.TrimRight(config.BaseURL, "/") + "/models"
	testResult := TestResult{Endpoint: endpoint}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return testResult, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: time.Duration(config.TimeoutSeconds) * time.Second}
	client.CheckRedirect = sameOriginRedirect(req.URL)
	response, err := client.Do(req)
	if err != nil {
		return testResult, classifyNetworkError(err)
	}
	defer response.Body.Close()
	testResult.HTTPStatus = response.StatusCode
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		testResult.Message = "鉴权失败，请检查 API Key"
		return testResult, nil
	}
	if response.StatusCode == http.StatusTooManyRequests {
		testResult.Message = "Provider 请求受限，请稍后重试"
		return testResult, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		testResult.Message = fmt.Sprintf("Provider 返回 HTTP %d", response.StatusCode)
		return testResult, nil
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 2<<20))
	if err := decoder.Decode(&payload); err != nil {
		testResult.Message = "响应不是兼容的模型列表"
		return testResult, nil
	}
	for _, model := range payload.Data {
		if model.ID != "" {
			testResult.Models = append(testResult.Models, model.ID)
			if model.ID == config.Model {
				testResult.ModelFound = true
			}
		}
	}
	sort.Strings(testResult.Models)
	testResult.OK = testResult.ModelFound
	if !testResult.ModelFound {
		testResult.Message = "连接成功，但模型列表中未找到配置的模型"
		return testResult, nil
	}
	capabilityOK, capabilityMessage := s.testToolCapability(ctx, client, config, key)
	testResult.CapabilityOK = capabilityOK
	testResult.OK = capabilityOK
	testResult.Message = capabilityMessage
	config.CapabilityOK = capabilityOK
	data, _ := json.MarshalIndent(config, "", "  ")
	_ = s.store.SaveProvider(config.ID, config.Enabled, string(data))
	return testResult, nil
}

// CompleteJSON is intentionally narrow: only trusted Agent code can supply the
// prompts, and the response must be a JSON object. It exposes no HTTP client to
// the model or caller.
func (s *Service) CompleteJSON(ctx context.Context, systemPrompt, userPayload string) (string, string, error) {
	config, err := s.Get()
	if err != nil {
		return "", "", err
	}
	if !config.Enabled || !config.CapabilityOK {
		return "", "", errors.New("Cleaning Agent provider has not passed capability testing")
	}
	key, _ := s.secrets.Load(config.ID)
	payload := map[string]any{
		"model": config.Model, "temperature": 0, "max_tokens": config.MaxOutputTokens,
		"response_format": map[string]string{"type": "json_object"},
		"messages":        []map[string]string{{"role": "system", "content": systemPrompt}, {"role": "user", "content": userPayload}},
	}
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: time.Duration(config.TimeoutSeconds) * time.Second, CheckRedirect: sameOriginRedirect(request.URL)}
	response, err := client.Do(request)
	if err != nil {
		return "", "", classifyNetworkError(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&result); err != nil {
		return "", "", errors.New("provider response is not OpenAI-compatible JSON")
	}
	if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", "", errors.New("provider returned an empty response")
	}
	return result.Choices[0].Message.Content, config.Model, nil
}

func (s *Service) testToolCapability(ctx context.Context, client *http.Client, config Config, key string) (bool, string) {
	payload := map[string]any{
		"model": config.Model, "max_tokens": 32, "temperature": 0,
		"messages":    []map[string]string{{"role": "system", "content": "Call the supplied read-only tool. Do not answer with prose."}, {"role": "user", "content": "Read the virtual disk overview."}},
		"tools":       []map[string]any{{"type": "function", "function": map[string]any{"name": "get_disk_overview", "description": "Return a built-in virtual disk sample", "parameters": map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}}}},
		"tool_choice": map[string]any{"type": "function", "function": map[string]string{"name": "get_disk_overview"}},
	}
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(config.BaseURL, "/") + "/chat/completions"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, "无法创建能力测试请求"
	}
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	response, err := client.Do(request)
	if err != nil {
		return false, "连接成功，但工具调用能力测试失败"
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Sprintf("连接成功，但工具能力测试返回 HTTP %d", response.StatusCode)
	}
	var result struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&result); err != nil {
		return false, "连接成功，但工具能力响应格式不兼容"
	}
	if len(result.Choices) == 0 || len(result.Choices[0].Message.ToolCalls) == 0 || result.Choices[0].Message.ToolCalls[0].Function.Name != "get_disk_overview" {
		return false, "模型可用，但不支持 Cleaning Agent 所需的结构化工具调用"
	}
	var arguments map[string]any
	if err := json.Unmarshal([]byte(result.Choices[0].Message.ToolCalls[0].Function.Arguments), &arguments); err != nil {
		return false, "模型工具参数不是有效 JSON"
	}
	if len(arguments) != 0 {
		return false, "模型为无参数工具生成了未知字段"
	}
	return true, "连接及工具能力测试通过，Cleaning Agent 可用"
}

func validateInput(input ConfigInput) (ConfigInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "Cleaning Agent"
	}
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Model = strings.TrimSpace(input.Model)
	if input.BaseURL == "" || input.Model == "" {
		return input, errors.New("base_url and model are required")
	}
	parsed, err := url.Parse(input.BaseURL)
	if err != nil || parsed.Host == "" {
		return input, errors.New("base_url is invalid")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return input, errors.New("remote providers require HTTPS; HTTP is allowed only for loopback addresses")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return input, errors.New("base_url cannot contain credentials, query parameters, or fragments")
	}
	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 60
	}
	if input.TimeoutSeconds < 10 || input.TimeoutSeconds > 300 {
		return input, errors.New("timeout_seconds must be between 10 and 300")
	}
	if input.MaxOutputTokens == 0 {
		input.MaxOutputTokens = 4096
	}
	if input.MaxOutputTokens < 256 || input.MaxOutputTokens > 32768 {
		return input, errors.New("max_output_tokens must be between 256 and 32768")
	}
	return input, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func sameOriginRedirect(origin *url.URL) func(*http.Request, []*http.Request) error {
	return func(next *http.Request, _ []*http.Request) error {
		if !strings.EqualFold(next.URL.Scheme, origin.Scheme) || !strings.EqualFold(next.URL.Host, origin.Host) {
			return errors.New("provider redirect changed origin")
		}
		return nil
	}
}
func classifyNetworkError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && errors.Is(urlErr.Err, context.DeadlineExceeded) {
		return errors.New("provider connection timed out")
	}
	return fmt.Errorf("provider connection failed: %w", err)
}
