package llm

import (
	"testing"

	"github.com/Kartik-2239/lightcode/internal/server/oauth"
)

func TestPreferredCopilotEndpointPrefersChat(t *testing.T) {
	got := preferredCopilotEndpoint([]string{"/responses", "/chat/completions"})
	if got != "/chat/completions" {
		t.Fatalf("expected chat completions endpoint, got %q", got)
	}
}

func TestPreferredCopilotEndpointFallsBackToResponses(t *testing.T) {
	got := preferredCopilotEndpoint([]string{"/responses", "ws:/responses"})
	if got != "/responses" {
		t.Fatalf("expected responses endpoint, got %q", got)
	}
}

func TestCopilotModelIDUsesID(t *testing.T) {
	got := copilotModelID(oauth.CopilotModel{Name: "GPT-5.5", ID: "gpt-5.5"})
	if got != "gpt-5.5" {
		t.Fatalf("expected model ID, got %q", got)
	}
}

func TestCopilotResponseTextJoinsOutputText(t *testing.T) {
	got := copilotResponseText(oauth.CopilotResponsesResponse{
		Output: []oauth.CopilotResponsesOutputItem{
			{Content: []oauth.CopilotResponsesOutputContent{{Text: "one"}}},
			{Content: []oauth.CopilotResponsesOutputContent{{Text: "two"}}},
		},
	})
	if got != "one\ntwo" {
		t.Fatalf("expected joined text, got %q", got)
	}
}
