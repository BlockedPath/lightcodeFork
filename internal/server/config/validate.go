package config

import (
	"context"
	"errors"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// ValidateProviderKey makes a minimal chat-completion request (max_tokens: 1) to
// the given provider to confirm the API key is accepted. It returns nil if the
// request succeeds. A short timeout keeps onboarding responsive.
func ValidateProviderKey(providerName, apiKey string) error {
	p, ok := ProviderByName(providerName)
	if !ok {
		return errors.New("unknown provider")
	}
	if len(p.Models) == 0 {
		return errors.New("provider has no models configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(p.BaseUrl))
	_, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     p.Models[0],
		Messages:  []openai.ChatCompletionMessageParamUnion{openai.UserMessage("hi")},
		MaxTokens: openai.Int(1),
	})
	return err
}
