// claude.go
package main

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type ClaudeClient struct {
	client *anthropic.Client
}

func NewClaudeClient(apiKey string) *ClaudeClient {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)
	return &ClaudeClient{
		client: client,
	}
}

func (c *ClaudeClient) SendMessage(systemPrompt, userPrompt string) (string, error) {
	ctx := context.Background()

	message, err := c.client.Messages.New(
		ctx,
		anthropic.MessageNewParams{
			Model:     anthropic.F(anthropic.ModelClaude3_5SonnetLatest),
			MaxTokens: anthropic.F(int64(4096)),
			System: anthropic.F([]anthropic.TextBlockParam{
				anthropic.NewTextBlock(systemPrompt),
			}),
			Messages: anthropic.F([]anthropic.MessageParam{
				anthropic.NewUserMessage(
					anthropic.NewTextBlock(userPrompt),
				),
			}),
		},
	)

	if err != nil {
		return "", fmt.Errorf("claude api error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from claude")
	}

	// Extract text from the first content block
	return message.Content[0].Text, nil
}
