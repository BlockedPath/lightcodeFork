package oauth_test

import (
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/db/models"
	"github.com/Kartik-2239/lightcode/internal/server/oauth"
	"github.com/Kartik-2239/lightcode/internal/server/tools"
)

func TestMakeOauthRequest(t *testing.T) {
	messages := []models.Message{
		{
			Data: models.EncodeMessageData(models.StoredMessageData{
				Role:    "user",
				Content: "Hello, how are you?",
			}),
		},
	}

	response, err := oauth.MakeOauthRequest("openai", "gpt-5.5", messages, "You are a helpful assistant.", tools.GetAllTools())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(response.Choices) == 0 {
		t.Fatalf("expected at least one choice, got none")
	}

	if response.Choices[0].Message.Content == "" {
		t.Fatalf("Expected response text, got empty string")
	}
}
