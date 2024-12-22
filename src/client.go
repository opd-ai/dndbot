package dndbot

import (
	"net/http"
)

type Client interface {
	SendMessage(systemPrompt, userPrompt string) (string, error)
}

type LLMClient struct {
	//client     *anthropic.Client
	http.Client
	apiKey string
}

func NewLLMClient(apiKey string) *LLMClient {
	return &LLMClient{
		Client: *http.DefaultClient,
		apiKey: apiKey,
	}
}

func (c *LLMClient) SendMessage(systemPrompt, userPrompt string) (string, error) {
	panic("NOT IMPLEMENTENTED UNTIL I CAN SELFHOST")
}
