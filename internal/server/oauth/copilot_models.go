package oauth

type CopilotModel struct {
	ID                  string                   `json:"id"`
	ModelPickerCategory string                   `json:"model_picker_category"`
	ModelPickerEnabled  bool                     `json:"model_picker_enabled"`
	Name                string                   `json:"name"`
	Object              string                   `json:"object"`
	Policy              CopilotModelPolicy       `json:"policy"`
	Preview             bool                     `json:"preview"`
	SupportedEndpoints  []string                 `json:"supported_endpoints"`
	Vendor              string                   `json:"vendor"`
	Version             string                   `json:"version"`
	Capabilities        CopilotModelCapabilities `json:"capabilities"`
}

type CopilotModelPolicy struct {
	State string `json:"state"`
	Terms string `json:"terms"`
}

type CopilotModelCapabilities struct {
	Family    string               `json:"family"`
	Limits    CopilotModelLimits   `json:"limits"`
	Object    string               `json:"object"`
	Supports  CopilotModelSupports `json:"supports"`
	Tokenizer string               `json:"tokenizer"`
	Type      string               `json:"type"`
}

type CopilotModelLimits struct {
	MaxContextWindowTokens      int                      `json:"max_context_window_tokens"`
	MaxNonStreamingOutputTokens int                      `json:"max_non_streaming_output_tokens"`
	MaxOutputTokens             int                      `json:"max_output_tokens"`
	MaxPromptTokens             int                      `json:"max_prompt_tokens"`
	Vision                      CopilotModelVisionLimits `json:"vision"`
}

type CopilotModelVisionLimits struct {
	MaxPromptImageSize  int      `json:"max_prompt_image_size"`
	MaxPromptImages     int      `json:"max_prompt_images"`
	SupportedMediaTypes []string `json:"supported_media_types"`
}

type CopilotModelSupports struct {
	AdaptiveThinking  bool     `json:"adaptive_thinking"`
	MaxThinkingBudget int      `json:"max_thinking_budget"`
	MinThinkingBudget int      `json:"min_thinking_budget"`
	ParallelToolCalls bool     `json:"parallel_tool_calls"`
	ReasoningEffort   []string `json:"reasoning_effort"`
	Streaming         bool     `json:"streaming"`
	StructuredOutputs bool     `json:"structured_outputs"`
	ToolCalls         bool     `json:"tool_calls"`
	Vision            bool     `json:"vision"`
}
