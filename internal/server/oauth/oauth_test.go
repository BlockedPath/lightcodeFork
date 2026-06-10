package oauth_test

import (
	"testing"
	"time"

	"github.com/Kartik-2239/lightcode/internal/server/oauth"
)

func TestStartAuthFlow(t *testing.T) {
	resp, err := oauth.StartAuthFlow()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == (oauth.DeviceCodeResponse{}) {
		t.Fatalf("Expected non-empty response, got empty string")
	}
	t.Log(resp)

	for i := 0; i < 25; i++ {
		resp, err := oauth.PollForAccessToken(resp.DeviceCode)
		if err != nil {
			t.Logf("Expected error while polling for access token, got %v", err)
		}
		if resp != (oauth.AuthResponse{}) && resp.AccessToken != "" {
			t.Logf("Received access token: %s", resp.AccessToken)
			break
		}
		time.Sleep(1 * time.Second)
	}

}

// func TestMakeOauthRequest(t *testing.T) {
// 	messages := []models.Message{
// 		{
// 			Data: models.EncodeMessageData(models.StoredMessageData{
// 				Role:    "user",
// 				Content: "Hello, how are you?",
// 			}),
// 		},
// 	}

// 	response, err := oauth.MakeOauthRequest("openai", "gpt-5.5", messages, "You are a helpful assistant.", tools.GetAllTools())
// 	if err != nil {
// 		t.Fatalf("Expected no error, got %v", err)
// 	}

// 	if len(response.Choices) == 0 {
// 		t.Fatalf("expected at least one choice, got none")
// 	}

// 	if response.Choices[0].Message.Content == "" {
// 		t.Fatalf("Expected response text, got empty string")
// 	}
// }
