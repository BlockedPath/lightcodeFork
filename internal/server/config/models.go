package config

func getDefaultProviders() []Provider {
	providers := []Provider{}

	anthropic_baseurl := "https://api.anthropic.com/v1"
	anthropic_models := []string{
		"claude-opus-4-7",
		"claude-opus-4-6",
		"claude-sonnet-4-6",
		"claude-opus-4-5",
		"claude-sonnet-4-5",
	}

	openai_baseurl := "https://api.openai.com/v1"
	openai_models := []string{
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.3-codex",
		"gpt-5.3",
	}

	providers = append(providers, Provider{
		BaseUrl: anthropic_baseurl,
		ApiKey:  "",
		Models:  anthropic_models,
	})
	providers = append(providers, Provider{
		BaseUrl: openai_baseurl,
		ApiKey:  "",
		Models:  openai_models,
	})

	return providers
}
