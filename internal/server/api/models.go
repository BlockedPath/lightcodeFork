package api

type ModelInfo struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	ApiKey   string `json:"api_key"`
	BaseUrl  string `json:"base_url"`
	LastUsed int64  `json:"last_used"`
}
type ModelTypes struct {
	Models []ModelInfo `json:"models"`
	Recent []ModelInfo `json:"recent_models"`
}
