package provider

type ConfigInput struct {
	Name            string `json:"name"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	APIKey          string `json:"api_key"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

type Config struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	CredentialRef   string `json:"credential_ref"`
	KeySaved        bool   `json:"key_saved"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	Enabled         bool   `json:"enabled"`
	CapabilityOK    bool   `json:"capability_ok"`
}

type TestResult struct {
	OK           bool     `json:"ok"`
	Message      string   `json:"message"`
	ModelFound   bool     `json:"model_found"`
	Models       []string `json:"models"`
	Endpoint     string   `json:"endpoint"`
	HTTPStatus   int      `json:"http_status"`
	CapabilityOK bool     `json:"capability_ok"`
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Completion struct {
	Content  string
	ToolCall []ToolCall
}
